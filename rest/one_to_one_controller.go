package rest

import (
	"net/http"

	"github.com/go-carrot/response"
	"github.com/go-carrot/surf"
	"github.com/go-carrot/turf"
	"github.com/go-carrot/validator"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/guregu/null.v3"
)

type OneToOneController struct {
	NestedModelNameSingular string
	ForeignReference        string
	GetBaseModel            surf.BuildModel
	GetNestedModel          surf.BuildModel
	LifecycleHooks          LifecycleHooks
	MethodWhiteList         []string
}

func (c OneToOneController) Register(r *httprouter.Router, mw turf.Middleware) {
	baseModelName := c.GetBaseModel().GetConfiguration().TableName
	nestedModelName := c.NestedModelNameSingular
	hasWhitelist := len(c.MethodWhiteList) != 0

	if !hasWhitelist || contains(c.MethodWhiteList, turf.CREATE) {
		r.HandlerFunc(http.MethodPost, "/"+baseModelName+"/:id/"+nestedModelName, mw(c.Create))
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.SHOW) {
		r.HandlerFunc(http.MethodGet, "/"+baseModelName+"/:id/"+nestedModelName, mw(c.Show))
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.UPDATE) {
		r.HandlerFunc(http.MethodPut, "/"+baseModelName+"/:id/"+nestedModelName, mw(c.Update))
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.DELETE) {
		r.HandlerFunc(http.MethodDelete, "/"+baseModelName+"/:id/"+nestedModelName, mw(c.Delete))
	}
}

func (c OneToOneController) Create(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Create nested model
	nestedModel := c.GetNestedModel()

	// Generate values to be tested
	values := getInsertValues(r, nestedModel)
	var id int64
	values = append(values, baseModelIdValue(&id, r))

	// Test values
	err := validator.Validate(values)
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Set ID
	model := c.GetBaseModel()
	configuration := model.GetConfiguration()
	for _, field := range configuration.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}

	// Load
	err = model.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Make sure it's not already set
	foreignId := int64(0)
	for _, field := range configuration.Fields {
		if field.Name == c.ForeignReference {
			switch v := field.Pointer.(type) {
			case *null.Int:
				foreignId = v.Int64
				break
			case *int64:
				foreignId = *v
				break
			}
		}
	}
	if foreignId != 0 {
		resp.SetResult(http.StatusConflict, nil)
		return
	}

	// Before Create hook
	if c.LifecycleHooks.BeforeCreate != nil {
		err := c.LifecycleHooks.BeforeCreate(resp, r, nestedModel)
		if err != nil {
			return
		}
	}

	// Create nested model
	err = nestedModel.Insert()
	if err != nil {
		handleInsertUpdateError(resp, err)
		return
	}

	// Get FK
	nestedModelId := int64(0)
	for _, field := range nestedModel.GetConfiguration().Fields {
		if field.Name == "id" {
			nestedModelId = *field.Pointer.(*int64)
			break
		}
	}

	// Get foreign ID
	for _, field := range model.GetConfiguration().Fields {
		if field.Name == c.ForeignReference {
			switch v := field.Pointer.(type) {
			case *null.Int:
				v.Int64 = nestedModelId
				v.Valid = true
				break
			case *int64:
				*v = nestedModelId
				break
			}
		}
	}

	// Update
	err = model.Update()
	if err != nil {
		resp.SetResult(http.StatusInternalServerError, nil)
		return
	}

	// After Create hook
	if c.LifecycleHooks.AfterCreate != nil {
		err := c.LifecycleHooks.AfterCreate(resp, r, nestedModel)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nestedModel)
}

func (c OneToOneController) Index(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// There is no index for a 1:1 model
	resp.SetResult(http.StatusMethodNotAllowed, nil)
}

func (c OneToOneController) Show(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Validate Params
	var id int64
	err := validator.Validate([]*validator.Value{
		baseModelIdValue(&id, r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Set ID
	model := c.GetBaseModel()
	configuration := model.GetConfiguration()
	for _, field := range configuration.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}

	// Load
	err = model.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Get foreign ID
	foreignId := int64(0)
	for _, field := range configuration.Fields {
		if field.Name == c.ForeignReference {
			switch v := field.Pointer.(type) {
			case *null.Int:
				foreignId = v.Int64
				break
			case *int64:
				foreignId = *v
				break
			}
		}
	}
	if foreignId == 0 {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Load nested model
	nestedModel := c.GetNestedModel()
	nestedConfig := nestedModel.GetConfiguration()
	for _, field := range nestedConfig.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = foreignId
			break
		}
	}

	// Before Show hook
	if c.LifecycleHooks.BeforeShow != nil {
		err := c.LifecycleHooks.BeforeShow(resp, r, nestedModel)
		if err != nil {
			return
		}
	}

	// Load
	err = nestedModel.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// After Show hook
	if c.LifecycleHooks.AfterShow != nil {
		err := c.LifecycleHooks.AfterShow(resp, r, model)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nestedModel)
}

func (c OneToOneController) Update(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Validate Params
	var id int64
	err := validator.Validate([]*validator.Value{
		baseModelIdValue(&id, r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Set ID
	model := c.GetBaseModel()
	configuration := model.GetConfiguration()
	for _, field := range configuration.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}

	// Load
	err = model.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Get foreign ID
	foreignId := int64(0)
	for _, field := range configuration.Fields {
		if field.Name == c.ForeignReference {
			switch v := field.Pointer.(type) {
			case *null.Int:
				foreignId = v.Int64
				break
			case *int64:
				foreignId = *v
				break
			}
		}
	}
	if foreignId == 0 {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Load nested model
	nestedModel := c.GetNestedModel()
	nestedConfig := nestedModel.GetConfiguration()
	for _, field := range nestedConfig.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = foreignId
			break
		}
	}

	// Load
	err = nestedModel.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Check `If-Unmodified-Since` header
	if !isUnmodifiedSinceHeader(nestedModel, r) {
		resp.SetErrorDetails("The `If-Unmodified-Since` condition is not satisfied")
		resp.SetResult(http.StatusPreconditionFailed, nil)
		return
	}

	// Generate + test values
	values := getUpdateValues(r, nestedModel)
	err = validator.Validate(values)
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Before Update hook
	if c.LifecycleHooks.BeforeUpdate != nil {
		err := c.LifecycleHooks.BeforeUpdate(resp, r, nestedModel)
		if err != nil {
			return
		}
	}

	// Update
	err = nestedModel.Update()
	if err != nil {
		handleInsertUpdateError(resp, err)
		return
	}

	// After Update hook
	if c.LifecycleHooks.AfterUpdate != nil {
		err := c.LifecycleHooks.AfterUpdate(resp, r, nestedModel)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nestedModel)
}

func (c OneToOneController) Delete(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Validate Params
	var id int64
	err := validator.Validate([]*validator.Value{
		baseModelIdValue(&id, r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Set ID
	model := c.GetBaseModel()
	configuration := model.GetConfiguration()
	for _, field := range configuration.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}

	// Load
	err = model.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Get foreign ID
	foreignId := int64(0)
	for _, field := range configuration.Fields {
		if field.Name == c.ForeignReference {
			switch v := field.Pointer.(type) {
			case *null.Int:
				// Get foreign ID to load foreign model
				foreignId = v.Int64

				break
			default:
				resp.SetErrorDetails(
					model.GetConfiguration().TableName +
						"." +
						c.ForeignReference +
						" is not nullable.  DELETE should not be allowed.",
				)
				resp.SetResult(http.StatusInternalServerError, nil)
				return
			}
		}
	}
	if foreignId == 0 {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Set nested model's ID
	nestedModel := c.GetNestedModel()
	nestedConfig := nestedModel.GetConfiguration()
	for _, field := range nestedConfig.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = foreignId
			break
		}
	}

	// Null out foreign reference
	for _, field := range configuration.Fields {
		if field.Name == c.ForeignReference {
			pointer := field.Pointer.(*null.Int)
			pointer.Valid = false
		}
	}

	// Before Delete hook
	if c.LifecycleHooks.BeforeDelete != nil {
		err := c.LifecycleHooks.BeforeDelete(resp, r, nestedModel)
		if err != nil {
			return
		}
	}

	// Remove foreign reference
	err = model.Update()
	if err != nil {
		resp.SetResult(http.StatusInternalServerError, nil)
		return
	}

	// Delete the model
	err = nestedModel.Delete()
	if err != nil {
		resp.SetResult(http.StatusInternalServerError, nil)
		return
	}

	// After Delete hook
	if c.LifecycleHooks.AfterDelete != nil {
		err := c.LifecycleHooks.AfterDelete(resp, r)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nil)
}
