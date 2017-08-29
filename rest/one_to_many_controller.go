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

type OneToManyController struct {
	GetBaseModel           surf.BuildModel
	GetNestedModel         surf.BuildModel
	NestedForeignReference string
	BelongsTo              func(baseModel, nestedModel surf.Model) bool
	LifecycleHooks         LifecycleHooks
	MethodWhiteList        []string
}

func (c OneToManyController) Register(r *httprouter.Router, mw turf.Middleware) {
	baseModelTableName := c.GetBaseModel().GetConfiguration().TableName
	nestedModelTableName := c.GetNestedModel().GetConfiguration().TableName
	hasWhitelist := len(c.MethodWhiteList) != 0

	if !hasWhitelist || contains(c.MethodWhiteList, turf.CREATE) {
		r.HandlerFunc(
			http.MethodPost,
			"/"+baseModelTableName+"/:id/"+nestedModelTableName,
			mw(c.Create),
		)
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.INDEX) {
		r.HandlerFunc(
			http.MethodGet,
			"/"+baseModelTableName+"/:id/"+nestedModelTableName,
			mw(c.Index),
		)
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.SHOW) {
		r.HandlerFunc(
			http.MethodGet,
			"/"+baseModelTableName+"/:id/"+nestedModelTableName+"/:nested_id",
			mw(c.Show),
		)
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.UPDATE) {
		r.HandlerFunc(
			http.MethodPut,
			"/"+baseModelTableName+"/:id/"+nestedModelTableName+"/:nested_id",
			mw(c.Update),
		)
	}
	if !hasWhitelist || contains(c.MethodWhiteList, turf.DELETE) {
		r.HandlerFunc(
			http.MethodDelete,
			"/"+baseModelTableName+"/:id/"+nestedModelTableName+"/:nested_id",
			mw(c.Delete),
		)
	}
}

func (c OneToManyController) Create(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Create Model
	model := c.GetNestedModel()

	// Generate values to be tested
	values := getInsertValues(r, model, c.NestedForeignReference)
	var foreignID int64
	values = append(values, baseModelIdValue(&foreignID, r))

	// Test values
	err := validator.Validate(values)
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Set nested reference
	for _, field := range model.GetConfiguration().Fields {
		if field.Name == c.NestedForeignReference {
			switch v := field.Pointer.(type) {
			case *null.Int:
				v.Int64 = foreignID
				v.Valid = true
				break
			case *int64:
				*v = foreignID
				break
			}
		}
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
		handleInsertUpdateError(resp, err)
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

func (c OneToManyController) Index(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Create bulkFetchConfig model
	bulkFetchConfig := surf.BulkFetchConfig{}

	// Validate Params
	var id int64
	var sort string
	err := validator.Validate([]*validator.Value{
		baseModelIdValue(&id, r),
		defaultLimitValue(&bulkFetchConfig.Limit, r),
		defaultOffsetValue(&bulkFetchConfig.Offset, r),
		defaultSortValue(&sort, c.GetNestedModel().GetConfiguration(), r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Load Base Model
	baseModel := c.GetBaseModel()
	for _, field := range baseModel.GetConfiguration().Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}
	err = baseModel.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Consume sort query
	bulkFetchConfig.ConsumeSortQuery(sort)

	// Consume If-Modified-Since header
	applyModSinceHeader(&bulkFetchConfig, r)

	// Set where predicate
	bulkFetchConfig.Predicates = []surf.Predicate{
		surf.Predicate{
			Field:         c.NestedForeignReference,
			PredicateType: surf.WHERE_EQUAL,
			Values:        []interface{}{id},
		},
	}

	// Before Index hook
	if c.LifecycleHooks.BeforeIndex != nil {
		err := c.LifecycleHooks.BeforeIndex(resp, r, &bulkFetchConfig)
		if err != nil {
			return
		}
	}

	// Fetch the models
	models, err := c.GetNestedModel().BulkFetch(bulkFetchConfig, c.GetNestedModel)
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusInternalServerError, nil)
		return
	}

	// After Index hook
	if c.LifecycleHooks.AfterIndex != nil {
		err := c.LifecycleHooks.AfterIndex(resp, r, &models)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, models)
}

func (c OneToManyController) Show(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Validate Params
	var id, nestedId int64
	err := validator.Validate([]*validator.Value{
		baseModelIdValue(&id, r),
		nestedModelIdValue(&nestedId, r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Load Base Model
	baseModel := c.GetBaseModel()
	for _, field := range baseModel.GetConfiguration().Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}
	err = baseModel.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Set ID
	nestedModel := c.GetNestedModel()
	for _, field := range nestedModel.GetConfiguration().Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = nestedId
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

	// Verify ownership
	if !c.BelongsTo(baseModel, nestedModel) {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// After Show hook
	if c.LifecycleHooks.AfterShow != nil {
		err := c.LifecycleHooks.AfterShow(resp, r, nestedModel)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nestedModel)
}

func (c OneToManyController) Update(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Validate Params
	var id, nestedId int64
	err := validator.Validate([]*validator.Value{
		baseModelIdValue(&id, r),
		nestedModelIdValue(&nestedId, r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Load Base Model
	baseModel := c.GetBaseModel()
	for _, field := range baseModel.GetConfiguration().Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}
	err = baseModel.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Load Nested Model
	nestedModel := c.GetNestedModel()
	for _, field := range nestedModel.GetConfiguration().Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = nestedId
			break
		}
	}
	err = nestedModel.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Verify ownership
	if !c.BelongsTo(baseModel, nestedModel) {
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
	values := getUpdateValues(r, nestedModel, c.NestedForeignReference)
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

func (c OneToManyController) Delete(w http.ResponseWriter, r *http.Request) {
	resp := response.New(w)
	defer resp.Output()

	// Validate Params
	var id, nestedId int64
	err := validator.Validate([]*validator.Value{
		baseModelIdValue(&id, r),
		nestedModelIdValue(&nestedId, r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Load Base Model
	baseModel := c.GetBaseModel()
	for _, field := range baseModel.GetConfiguration().Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = id
			break
		}
	}
	err = baseModel.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Load Nested Model
	nestedModel := c.GetNestedModel()
	for _, field := range nestedModel.GetConfiguration().Fields {
		if field.Name == "id" {
			*field.Pointer.(*int64) = nestedId
			break
		}
	}
	err = nestedModel.Load()
	if err != nil {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Verify ownership
	if !c.BelongsTo(baseModel, nestedModel) {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Before Delete hook
	if c.LifecycleHooks.BeforeDelete != nil {
		err := c.LifecycleHooks.BeforeDelete(resp, r, nestedModel)
		if err != nil {
			return
		}
	}

	// Delete
	err = nestedModel.Delete()
	if err != nil {
		resp.SetErrorDetails(err.Error())
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
