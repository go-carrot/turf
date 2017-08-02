package rest

import (
	"math"
	"net/http"

	"github.com/go-carrot/surf"
	"github.com/go-carrot/turf"
	"github.com/go-carrot/validator"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

type ManyToManyController struct {
	GetBaseModel                surf.BuildModel
	GetNestedModel              surf.BuildModel
	GetRelationModel            surf.BuildModel
	BaseModelForeignReference   string
	NestedModelForeignReference string
	LifecycleHooks              LifecycleHooks
	MethodWhiteList             []string
}

func (c ManyToManyController) Register(r *httprouter.Router, mw turf.Middleware) {
	baseModelTableName := c.GetBaseModel().GetConfiguration().TableName
	nestedModelTableName := c.GetNestedModel().GetConfiguration().TableName
	hasWhitelist := len(c.MethodWhiteList) != 0

	if !hasWhitelist || contains(c.MethodWhiteList, turf.CREATE) {
		r.HandlerFunc(
			http.MethodPost,
			"/"+baseModelTableName+"/:id/"+nestedModelTableName+"/:nested_id",
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
	if !hasWhitelist || contains(c.MethodWhiteList, turf.DELETE) {
		r.HandlerFunc(
			http.MethodDelete,
			"/"+baseModelTableName+"/:id/"+nestedModelTableName+"/:nested_id",
			mw(c.Delete),
		)
	}
}

func (c ManyToManyController) Create(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
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

	// Prep relation model
	relationModel := c.GetRelationModel()
	for _, field := range relationModel.GetConfiguration().Fields {
		if field.Name == c.BaseModelForeignReference {
			*field.Pointer.(*int64) = id
		} else if field.Name == c.NestedModelForeignReference {
			*field.Pointer.(*int64) = nestedId
		}
	}

	// Before Create hook
	if c.LifecycleHooks.BeforeCreate != nil {
		err := c.LifecycleHooks.BeforeCreate(resp, r, relationModel)
		if err != nil {
			return
		}
	}

	// Insert
	err = relationModel.Insert()
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
		err := c.LifecycleHooks.AfterCreate(resp, r, relationModel)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nil)
}

func (c ManyToManyController) Index(w http.ResponseWriter, r *http.Request) {
	// TODO, this can be more efficient with an IN predicate with a sub query
	resp := newRestResponse(&w, r)
	defer resp.Output()

	// Validate Params
	var id int64
	var limit, offset int
	var sort string
	err := validator.Validate([]*validator.Value{
		baseModelIdValue(&id, r),
		defaultLimitValue(&limit, r),
		defaultOffsetValue(&offset, r),
		defaultSortValue(&sort, c.GetNestedModel().GetConfiguration(), r),
	})
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusBadRequest, nil)
		return
	}

	// Verify base model exists
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

	// Load relations
	relations, err := c.GetRelationModel().BulkFetch(surf.BulkFetchConfig{
		Limit: int(math.MaxInt32),
		Predicates: []surf.Predicate{{
			Field:         c.BaseModelForeignReference,
			PredicateType: surf.WHERE_EQUAL,
			Values:        []interface{}{id},
		}},
	}, c.GetRelationModel)
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusInternalServerError, nil)
		return
	}
	if len(relations) == 0 {
		resp.SetResult(http.StatusOK, make([]interface{}, 0))
		return
	}

	// Get all the IDs
	ids := make([]interface{}, 0)
	for _, r := range relations {
		for _, field := range r.GetConfiguration().Fields {
			if field.Name == c.NestedModelForeignReference {
				ids = append(ids, *(field.Pointer.(*int64)))
			}
		}
	}

	// Prepare BulkFetchConfig
	fetchConfig := surf.BulkFetchConfig{
		Limit:  limit,
		Offset: offset,
		Predicates: []surf.Predicate{{
			Field:         "id",
			PredicateType: surf.WHERE_IN,
			Values:        ids,
		}},
	}
	fetchConfig.ConsumeSortQuery(sort)
	applyModSinceHeader(&fetchConfig, r)

	// Before Index hook
	if c.LifecycleHooks.BeforeIndex != nil {
		err := c.LifecycleHooks.BeforeIndex(resp, r, &fetchConfig)
		if err != nil {
			return
		}
	}

	// Load nested models
	nestedModels, err := c.GetNestedModel().BulkFetch(fetchConfig, c.GetNestedModel)
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusInternalServerError, nil)
		return
	}

	// After Index hook
	if c.LifecycleHooks.AfterIndex != nil {
		err := c.LifecycleHooks.AfterIndex(resp, r, &nestedModels)
		if err != nil {
			return
		}
	}

	// OK
	resp.SetResult(http.StatusOK, nestedModels)
}

func (c ManyToManyController) Show(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
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

	// Verify the relation exists
	relations, err := c.GetRelationModel().BulkFetch(surf.BulkFetchConfig{
		Limit: 1,
		Predicates: []surf.Predicate{
			{
				Field:         c.BaseModelForeignReference,
				PredicateType: surf.WHERE_EQUAL,
				Values:        []interface{}{id},
			},
			{
				Field:         c.NestedModelForeignReference,
				PredicateType: surf.WHERE_EQUAL,
				Values:        []interface{}{nestedId},
			},
		},
	}, c.GetRelationModel)
	if len(relations) < 1 {
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

	// Load nested model
	err = nestedModel.Load()
	if err != nil {
		resp.SetErrorDetails(err.Error())
		resp.SetResult(http.StatusInternalServerError, nil)
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

func (c ManyToManyController) Update(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
	defer resp.Output()

	// There is no update for a many:many model
	resp.SetResult(http.StatusMethodNotAllowed, nil)
}

func (c ManyToManyController) Delete(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
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

	// Verify the relation exists
	relations, err := c.GetRelationModel().BulkFetch(surf.BulkFetchConfig{
		Limit: 1,
		Predicates: []surf.Predicate{
			{
				Field:         c.BaseModelForeignReference,
				PredicateType: surf.WHERE_EQUAL,
				Values:        []interface{}{id},
			},
			{
				Field:         c.NestedModelForeignReference,
				PredicateType: surf.WHERE_EQUAL,
				Values:        []interface{}{nestedId},
			},
		},
	}, c.GetRelationModel)
	if len(relations) < 1 {
		resp.SetResult(http.StatusNotFound, nil)
		return
	}

	// Get relation
	relation := relations[0]

	// Before Delete hook
	if c.LifecycleHooks.BeforeDelete != nil {
		err := c.LifecycleHooks.BeforeDelete(resp, r, relation)
		if err != nil {
			return
		}
	}

	// Delete relation
	err = relation.Delete()
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
