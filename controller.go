package turf

import (
	"github.com/go-carrot/response"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

const (
	CREATE = "CREATE"
	INDEX  = "INDEX"
	SHOW   = "SHOW"
	UPDATE = "UPDATE"
	DELETE = "DELETE"
)

type Controller interface {
	Register(r *httprouter.Router, mw Middleware)
	Create(http.ResponseWriter, *http.Request)
	Index(w http.ResponseWriter, r *http.Request)
	Show(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

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

type Middleware func(next http.HandlerFunc) http.HandlerFunc
