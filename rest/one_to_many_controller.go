package rest

import (
	"github.com/go-carrot/surf"
	"github.com/go-carrot/turf"
	"github.com/go-carrot/validator"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type OneToManyController struct {
	GetBaseModel    surf.BuildModel
	GetNestedModel  surf.BuildModel
	BelongsTo       func(baseModel, nestedModel surf.Model) bool
	LifecycleHooks  turf.LifecycleHooks
	MethodWhiteList []string
}

func (c OneToManyController) Register(r *httprouter.Router, mw turf.Middleware) {
	/*
	baseModelTableName := c.GetBaseModel().GetConfiguration().TableName
	nestedModelTableName := c.GetNestedModel.GetConfiguration().TableName

	if len(c.MethodWhiteList) == 0 || contains(c.MethodWhiteList, turf.CREATE) {

		r.HandlerFunc(http.MethodPost, "/"+baseModelTableName+"/:id/"+, mw(c.Create))
	}
	if len(c.MethodWhiteList) == 0 || contains(c.MethodWhiteList, turf.INDEX) {
		r.HandlerFunc(http.MethodGet, "/"+tableName, mw(c.Index))
	}
	if len(c.MethodWhiteList) == 0 || contains(c.MethodWhiteList, turf.SHOW) {
		r.HandlerFunc(http.MethodGet, "/"+tableName+"/:id", mw(c.Show))
	}
	if len(c.MethodWhiteList) == 0 || contains(c.MethodWhiteList, turf.UPDATE) {
		r.HandlerFunc(http.MethodPut, "/"+tableName+"/:id", mw(c.Update))
	}
	if len(c.MethodWhiteList) == 0 || contains(c.MethodWhiteList, turf.DELETE) {
		r.HandlerFunc(http.MethodDelete, "/"+tableName+"/:id", mw(c.Delete))
	}
	*/
}

func (c OneToManyController) Create(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
	defer resp.Output()

	resp.SetResult(http.StatusNotImplemented, nil)
}

func (c OneToManyController) Index(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
	defer resp.Output()

	resp.SetResult(http.StatusNotImplemented, nil)
}

func (c OneToManyController) Show(w http.ResponseWriter, r *http.Request) {
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
			*field.Pointer.(*int64) = id
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

	// OK
	resp.SetResult(http.StatusOK, nestedModel)
}

func (c OneToManyController) Update(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
	defer resp.Output()

	resp.SetResult(http.StatusNotImplemented, nil)
}

func (c OneToManyController) Delete(w http.ResponseWriter, r *http.Request) {
	resp := newRestResponse(&w, r)
	defer resp.Output()

	resp.SetResult(http.StatusNotImplemented, nil)
}
