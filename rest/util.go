package rest

import (
	"net/http"
	"time"

	"github.com/go-carrot/surf"
	"github.com/guregu/null"
)

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// https://tools.ietf.org/html/rfc7232#section-3.3
//
// > A recipient MUST ignore the If-Modified-Since header field if the
// > received field-value is not a valid HTTP-date...
func applyModSinceHeader(bulkFetchConfig *surf.BulkFetchConfig, r *http.Request) {
	header := r.Header["If-Modified-Since"]
	if len(header) > 0 {
		lastModified, err := time.Parse(time.RFC1123, header[0])
		if err == nil {
			bulkFetchConfig.Predicates = append(bulkFetchConfig.Predicates, surf.Predicate{
				Field:         "modified_at",
				PredicateType: surf.WHERE_GREATER_THAN,
				Values:        []interface{}{lastModified},
			})
		}
	}
}

// https://tools.ietf.org/html/rfc7232#section-3.4
//
// > A recipient MUST ignore the If-Unmodified-Since header field if the
// > received field-value is not a valid HTTP-date.
func isUnmodifiedSinceHeader(model surf.Model, r *http.Request) bool {
	header := r.Header["If-Unmodified-Since"]
	if len(header) > 0 {
		// Parse If-Unmodified-Since to get the time.Time
		ifUnmodifiedSince, err := time.Parse(time.RFC1123, header[0])
		if err == nil {
			// Figure out the time the model was actually last modified
			var modifiedAt time.Time
			for _, field := range model.GetConfiguration().Fields {
				if field.Name == "modified_at" {
					switch v := field.Pointer.(type) {
					case *time.Time:
						modifiedAt = *v
					case *null.Time:
						modifiedAt = v.Time
					}
					break
				}
			}

			// Make sure we satisfy our `If-Unmodified-Since` condition.
			// If no `modified_at` attribute is set on the model, there is no
			// way we can determine that it's not been modified, so we must return
			// false and prevent the model from being updated.
			//
			// We also need to take a floor of the nsecs in the actual modified at date,
			// as RFC1123 is only specific up to seconds.
			if modifiedAt.IsZero() || floorNsecs(modifiedAt).After(ifUnmodifiedSince) {
				return false
			}
		}
	}
	return true
}

// floorNsecs
func floorNsecs(in time.Time) time.Time {
	return time.Date(
		in.Year(),
		in.Month(),
		in.Day(),
		in.Hour(),
		in.Minute(),
		in.Second(),
		0,
		in.Location(),
	)
}
