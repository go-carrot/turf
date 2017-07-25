package turf

import (
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

type Middleware func(next http.HandlerFunc) http.HandlerFunc

type Controller interface {
	Register(r *httprouter.Router, mw Middleware)
	Create(http.ResponseWriter, *http.Request)
	Index(w http.ResponseWriter, r *http.Request)
	Show(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}
