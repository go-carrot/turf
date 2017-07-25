package rest

import (
	"github.com/go-carrot/response"
	"github.com/go-carrot/surf"
	"net/http"
)

type BaseLifecycleHook func(response *response.Response, request *http.Request, model surf.Model) error

type AfterIndexLifecycleHook func(response *response.Response, request *http.Request, model *[]surf.Model) error

type BeforeIndexLifecycleHook func(response *response.Response, request *http.Request, model *surf.BulkFetchConfig) error

type AfterDeleteLifecycleHook func(response *response.Response, request *http.Request) error

type LifecycleHooks struct {
	// After validation and model prep, but before creation
	BeforeCreate BaseLifecycleHook

	// After create, but before HTTP response
	AfterCreate BaseLifecycleHook

	// After the BulkFetchConfig is set, but before the fetch
	BeforeIndex BeforeIndexLifecycleHook

	// After the fetch, but before the response
	AfterIndex AfterIndexLifecycleHook

	// After the model is prepared to be loaded, but before it is actually loaded
	BeforeShow BaseLifecycleHook

	// After the model is loaded, but before the HTTP response
	AfterShow BaseLifecycleHook

	// After the model has been loaded + the values have been validated + set to the model,
	// but before the model is updated
	BeforeUpdate BaseLifecycleHook

	// After update, but before HTTP response
	AfterUpdate BaseLifecycleHook

	// After the models identifier has been set, but before the deletion
	BeforeDelete BaseLifecycleHook

	// After the model has been deleted, but before the HTTP response
	AfterDelete AfterDeleteLifecycleHook
}
