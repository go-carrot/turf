package rest

import (
	"github.com/go-carrot/rules"
	"github.com/go-carrot/surf"
	"github.com/go-carrot/turf"
	"github.com/go-carrot/validator"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
	"gopkg.in/guregu/null.v3"
	"net/http"
)

type BaseController struct {
	GetModel        surf.BuildModel
	LifecycleHooks  turf.LifecycleHooks
	MethodWhiteList []string
}

func (c BaseController) Register(r *httprouter.Router, mw turf.Middleware) {
	tableName := c.GetModel().GetConfiguration().TableName
	hasWhitelist := len(c.MethodWhiteList) != 0

	if !hasWhitelist || contains(c.MethodWhiteList, turf.CREATE) {
		r.HandlerFunc(http.MethodPost, "/"+tableName, mw(c.Create))
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.INDEX) {
		r.HandlerFunc(http.MethodGet, "/"+tableName, mw(c.Index))
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.SHOW) {
		r.HandlerFunc(http.MethodGet, "/"+tableName+"/:id", mw(c.Show))
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.UPDATE) {
		r.HandlerFunc(http.MethodPut, "/"+tableName+"/:id", mw(c.Update))
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.DELETE) {
		r.HandlerFunc(http.MethodDelete, "/"+tableName+"/:id", mw(c.Delete))
	}
}

func (c BaseController) Create(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
	defer resp.Output()

	// Create Model
	model := c.GetModel()

	// Generate values
	var values []*validator.Value
	for _, field := range model.GetConfiguration().Fields {
		if field.Insertable && !field.SkipValidation {

			// Determine TypeHandler (if any)
			var typeHandler validator.TypeHandler
			switch field.Pointer.(type) {
			case *null.String:
				typeHandler = validator.NullStringHandler
			case *null.Int:
				typeHandler = validator.NullIntHandler
			case *null.Bool:
				typeHandler = validator.NullBoolHandler
			case *null.Time:
				typeHandler = validator.NullTimeHandler
			case *null.Float:
				typeHandler = validator.NullFloatHandler
			}

			// Set required for primitives
			var valueRules []validator.Rule
			if typeHandler == nil {
				valueRules = []validator.Rule{rules.IsSet}
			}

			// Generate value
			values = append(values,
				&validator.Value{
					Result:      field.Pointer,
					Name:        field.Name,
					Input:       r.FormValue(field.Name),
					Rules:       valueRules,
					TypeHandler: typeHandler,
				})
		}
	}

	// Test values
	err := validator.Validate(values)
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Before Create hook
	if c.LifecycleHooks.BeforeCreate != nil {
		err := c.LifecycleHooks.BeforeCreate(resp, r, model)
		if err != nil {
			return
		}
	}

	// Insert
	err = model.Insert()
	if err != nil {
		pqErr, isPqError := err.(*pq.Error)
		if isPqError {
			resp.SetErrorDetails(pqErr.Detail)
			switch pqErr.Code {
			case POSTGRES_NOT_NULL_VIOLATION:
			case POSTGRES_ERROR_FOREIGN_KEY_VIOLATION:
				resp.SetResult(http.StatusBadRequest, nil)
				return
			case POSTGRES_ERROR_UNIQUE_VIOLATION:
				resp.SetResult(http.StatusConflict, nil)
				return
			}
		}
		resp.SetResult(http.StatusInternalServerError, nil)
		return
	}

	// After Create hook
	if c.LifecycleHooks.AfterCreate != nil {
		err := c.LifecycleHooks.AfterCreate(resp, r, model)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, model)
}

func (c BaseController) Index(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
	defer resp.Output()

	// Create bulkFetchConfig model
	bulkFetchConfig := surf.BulkFetchConfig{}

	// Validate
	var sort string
	err := validator.Validate([]*validator.Value{
		defaultLimitValue(&bulkFetchConfig.Limit, r),
		defaultOffsetValue(&bulkFetchConfig.Offset, r),
		defaultSortValue(&sort, c.GetModel().GetConfiguration(), r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Consume sort query
	bulkFetchConfig.ConsumeSortQuery(sort)

	// Before Index hook
	if c.LifecycleHooks.BeforeIndex != nil {
		err := c.LifecycleHooks.BeforeIndex(resp, r, &bulkFetchConfig)
		if err != nil {
			return
		}
	}

	// Load models
	models, err := c.GetModel().BulkFetch(bulkFetchConfig, c.GetModel)
	if err != nil {
		resp.SetResult(http.StatusBadRequest, nil)
		resp.SetErrorDetails(err.Error())
		return
	}

	// After Index hook
	if c.LifecycleHooks.AfterIndex != nil {
		err := c.LifecycleHooks.AfterIndex(resp, r, &bulkFetchConfig)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, models)
}

func (c BaseController) Show(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
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
	model := c.GetModel()
	configuration := model.GetConfiguration()
	for _, field := range configuration.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}

	// Before Show hook
	if c.LifecycleHooks.BeforeShow != nil {
		err := c.LifecycleHooks.BeforeShow(resp, r, model)
		if err != nil {
			return
		}
	}

	// Load
	err = model.Load()
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
	resp.SetResult(http.StatusOK, model)
}

func (c BaseController) Update(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
	defer resp.Output()

	// Create Model
	model := c.GetModel()

	// Validate ID
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

	// Generate values
	var values []*validator.Value
	for _, field := range model.GetConfiguration().Fields {
		if field.Updatable && !field.SkipValidation {
			// Determine TypeHandler (if any)
			var typeHandler validator.TypeHandler
			switch field.Pointer.(type) {
			case *null.String:
				typeHandler = validator.NullStringHandler
			case *null.Int:
				typeHandler = validator.NullIntHandler
			case *null.Bool:
				typeHandler = validator.NullBoolHandler
			case *null.Time:
				typeHandler = validator.NullTimeHandler
			case *null.Float:
				typeHandler = validator.NullFloatHandler
			}

			// Generate value
			values = append(values,
				&validator.Value{
					Result:      field.Pointer,
					Name:        field.Name,
					Input:       r.FormValue(field.Name),
					TypeHandler: typeHandler,
				})
		}
	}

	// Test values
	err = validator.Validate(values)
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Before Update hook
	if c.LifecycleHooks.BeforeUpdate != nil {
		err := c.LifecycleHooks.BeforeUpdate(resp, r, model)
		if err != nil {
			return
		}
	}

	// Update
	err = model.Update()
	if err != nil {
		pqErr, isPqError := err.(*pq.Error)
		if isPqError {
			resp.SetErrorDetails(pqErr.Detail)
			switch pqErr.Code {
			case POSTGRES_NOT_NULL_VIOLATION:
			case POSTGRES_ERROR_FOREIGN_KEY_VIOLATION:
				resp.SetResult(http.StatusBadRequest, nil)
				return
			case POSTGRES_ERROR_UNIQUE_VIOLATION:
				resp.SetResult(http.StatusConflict, nil)
				return
			}
		}
		resp.SetResult(http.StatusInternalServerError, nil)
		return
	}

	// After Update hook
	if c.LifecycleHooks.AfterUpdate != nil {
		err := c.LifecycleHooks.AfterUpdate(resp, r, model)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, model)
}

func (c BaseController) Delete(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
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
	model := c.GetModel()
	configuration := model.GetConfiguration()
	for _, field := range configuration.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}

	// Before Delete hook
	if c.LifecycleHooks.BeforeDelete != nil {
		err := c.LifecycleHooks.BeforeDelete(resp, r, model)
		if err != nil {
			return
		}
	}

	// Delete
	err = model.Delete()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// After Delete hook
	if c.LifecycleHooks.AfterDelete != nil {
		err := c.LifecycleHooks.AfterDelete(resp, r, model)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nil)
}
