package rest

import (
	"net/http"
	"time"

	"github.com/go-carrot/surf"
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
