package rest

import (
	"net/http"

	"github.com/go-carrot/rules"
	"github.com/go-carrot/surf"
	"github.com/go-carrot/validator"
)

func getInsertValues(r *http.Request, model surf.Model, exclusions ...string) []*validator.Value {
	var values []*validator.Value
	for _, field := range model.GetConfiguration().Fields {
		if field.Insertable && !field.SkipValidation && !contains(exclusions, field.Name) {
			// Strings must be set, use null.String if you want empty string
			var valueRules []validator.Rule
			switch field.Pointer.(type) {
			case *string:
				valueRules = []validator.Rule{rules.MinLen(1)}
			}

			// Validate values
			values = append(values,
				&validator.Value{
					Result: field.Pointer,
					Name:   field.Name,
					Input:  r.FormValue(field.Name),
					Rules:  valueRules,
				})
		}
	}
	return values
}

func getUpdateValues(r *http.Request, model surf.Model, exclusions ...string) []*validator.Value {
	// Parse form
	if r.Form == nil {
		r.ParseMultipartForm(32 << 20)
	}

	// Get values
	var values []*validator.Value
	for _, field := range model.GetConfiguration().Fields {
		_, keySet := r.Form[field.Name]
		if keySet && field.Updatable && !field.SkipValidation && !contains(exclusions, field.Name) {
			// Strings must be set, use null.String if you want empty string
			var valueRules []validator.Rule
			switch field.Pointer.(type) {
			case *string:
				valueRules = []validator.Rule{rules.MinLen(1)}
			}

			// Generate value
			values = append(values,
				&validator.Value{
					Result: field.Pointer,
					Name:   field.Name,
					Input:  r.FormValue(field.Name),
					Rules:  valueRules,
				})
		}
	}
	return values
}
