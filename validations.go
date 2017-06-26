package turf

import (
	"github.com/go-carrot/validator"
	"net/http"
	"github.com/go-carrot/rules"
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
