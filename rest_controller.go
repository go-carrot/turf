package turf

import (
	"github.com/go-carrot/rules"
	"github.com/go-carrot/surf"
	"github.com/go-carrot/turf/response"
	"github.com/go-carrot/validator"
	"github.com/lib/pq"
	"gopkg.in/guregu/null.v3"
	"net/http"
)

const (
	POSTGRES_NOT_NULL_VIOLATION          = "23502"
	POSTGRES_ERROR_FOREIGN_KEY_VIOLATION = "23503"
	POSTGRES_ERROR_UNIQUE_VIOLATION      = "23505"
)

type RestController struct {
	GetModel       surf.BuildModel
	LifecycleHooks LifecycleHooks
}

func (rc RestController) Create(w http.ResponseWriter, r *http.Request) {
	resp := response.New(&w, r)
	defer resp.Output()

	// Create Model
	model := rc.GetModel()

	// Generate values
	var values []*validator.Value
	for _, field := range model.GetConfiguration().Fields {
		if field.Insertable {

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
	if rc.LifecycleHooks.BeforeCreate != nil {
		err := rc.LifecycleHooks.BeforeCreate(resp, r, &model)
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
	if rc.LifecycleHooks.AfterCreate != nil {
		err := rc.LifecycleHooks.AfterCreate(resp, r, &model)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, model)
}

func (rc RestController) Index(w http.ResponseWriter, r *http.Request) {
	resp := response.New(&w, r)
	defer resp.Output()

	// Create bulkFetchConfig model
	bulkFetchConfig := surf.BulkFetchConfig{}

	// Validate
	var sort string
	err := validator.Validate([]*validator.Value{
		{
			Result: &bulkFetchConfig.Limit,
			Name:   "limit",
			Input:  r.URL.Query().Get("limit"),
			Rules: []validator.Rule{
				rules.MinValue(1),
			},
			Default: "10",
		},
		{
			Result: &bulkFetchConfig.Offset,
			Name:   "offset",
			Input:  r.URL.Query().Get("offset"),
			Rules: []validator.Rule{
				rules.MinValue(0),
			},
			Default: "0",
		},
		{
			Result:  &sort,
			Name:    "sort",
			Input:   r.URL.Query().Get("sort"),
			Rules:   []validator.Rule{validateSortField(rc.GetModel().GetConfiguration())},
			Default: "created_at",
		},
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Consume sort query
	bulkFetchConfig.ConsumeSortQuery(sort)

	// Before Index hook
	if rc.LifecycleHooks.BeforeIndex != nil {
		err := rc.LifecycleHooks.BeforeIndex(resp, r, &bulkFetchConfig)
		if err != nil {
			return
		}
	}

	// Load models
	models, err := rc.GetModel().BulkFetch(bulkFetchConfig, rc.GetModel)
	if err != nil {
		resp.SetResult(http.StatusBadRequest, nil)
		resp.SetErrorDetails(err.Error())
		return
	}

	// After Index hook
	if rc.LifecycleHooks.AfterIndex != nil {
		err := rc.LifecycleHooks.AfterIndex(resp, r, &bulkFetchConfig)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, models)
}

func (rc RestController) Show(w http.ResponseWriter, r *http.Request) {
	resp := response.New(&w, r)
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
	model := rc.GetModel()
	configuration := model.GetConfiguration()
	for _, field := range configuration.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}

	// Before Show hook
	if rc.LifecycleHooks.BeforeShow != nil {
		err := rc.LifecycleHooks.BeforeShow(resp, r, &model)
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
	if rc.LifecycleHooks.AfterShow != nil {
		err := rc.LifecycleHooks.AfterShow(resp, r, &model)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, model)
}

func (rc RestController) Update(w http.ResponseWriter, r *http.Request) {
	resp := response.New(&w, r)
	defer resp.Output()

	// Create Model
	model := rc.GetModel()

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
		if field.Updatable {
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
	if rc.LifecycleHooks.BeforeUpdate != nil {
		err := rc.LifecycleHooks.BeforeUpdate(resp, r, &model)
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
	if rc.LifecycleHooks.AfterUpdate != nil {
		err := rc.LifecycleHooks.AfterUpdate(resp, r, &model)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, model)
}

func (rc RestController) Delete(w http.ResponseWriter, r *http.Request) {
	resp := response.New(&w, r)
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
	model := rc.GetModel()
	configuration := model.GetConfiguration()
	for _, field := range configuration.Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}

	// Before Delete hook
	if rc.LifecycleHooks.BeforeDelete != nil {
		err := rc.LifecycleHooks.BeforeDelete(resp, r, &model)
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
	if rc.LifecycleHooks.AfterDelete != nil {
		err := rc.LifecycleHooks.AfterDelete(resp, r, &model)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nil)
}
