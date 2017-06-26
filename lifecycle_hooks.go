package turf

import (
	"github.com/go-carrot/response"
	"github.com/go-carrot/surf"
	"net/http"
)

type LifecycleHook func(response *response.Response, request *http.Request, worker *surf.Worker) error

type LifecycleHooks struct {
	BeforeCreate LifecycleHook
	AfterCreate  LifecycleHook
	BeforeShow   LifecycleHook
	AfterShow    LifecycleHook
	BeforeUpdate LifecycleHook
	AfterUpdate  LifecycleHook
	BeforeDelete LifecycleHook
	AfterDelete  LifecycleHook
}
