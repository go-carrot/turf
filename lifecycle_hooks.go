package turf

import (
	"github.com/go-carrot/response"
	"net/http"
)

type LifecycleHook func(response *response.Response, request *http.Request, model interface{}) error

type LifecycleHooks struct {
	BeforeCreate LifecycleHook
	AfterCreate  LifecycleHook
	BeforeShow   LifecycleHook
	AfterShow    LifecycleHook
	BeforeIndex  LifecycleHook
	AfterIndex   LifecycleHook
	BeforeUpdate LifecycleHook
	AfterUpdate  LifecycleHook
	BeforeDelete LifecycleHook
	AfterDelete  LifecycleHook
}
