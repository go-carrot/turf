package rest

import (
	"fmt"
	"github.com/go-carrot/rules"
	"github.com/go-carrot/surf"
	"github.com/go-carrot/validator"
	"net/http"
	"strings"
)

func baseModelIdValue(output *int64, r *http.Request) *validator.Value {
	return &validator.Value{
		Result: output,
		Name:   "id",
		Input:  strings.Split(r.URL.Path, "/")[2],
		Rules: []validator.Rule{
			rules.IsSet,
		},
	}
}

func nestedModelIdValue(output *int64, r *http.Request) *validator.Value {
	return &validator.Value{
		Result: output,
		Name:   "nested_id",
		Input:  strings.Split(r.URL.Path, "/")[4],
		Rules: []validator.Rule{
			rules.IsSet,
		},
	}
}

func defaultLimitValue(output *int64, r *http.Request) *validator.Value {
	return &validator.Value{
		Result:  output,
		Name:    "limit",
		Input:   r.URL.Query().Get("limit"),
		Default: "20",
		Rules: []validator.Rule{
			rules.MinValue(1),
			rules.MaxValue(5000),
		},
	}
}

func defaultOffsetValue(output *int64, r *http.Request) *validator.Value {
	return &validator.Value{
		Result:  output,
		Name:    "offset",
		Input:   r.URL.Query().Get("offset"),
		Default: "0",
		Rules: []validator.Rule{
			rules.MinValue(0),
		},
	}
}

func validateSortField(config *surf.Configuration) func(name, input string) error {
	return func(name string, input string) error {
	SortBys:
		for _, inputSort := range strings.Split(input, ",") {
			for _, field := range config.Fields {
				if inputSort == field.Name || inputSort == "-"+field.Name {
					continue SortBys
				}
			}
			return fmt.Errorf("Parameter '%v' must only contain fields within the model. Input '%v' is invalid.", name, inputSort)
		}
		return nil
	}
}
