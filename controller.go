package turf

import (
	"github.com/go-carrot/surf"
	"net/http"
)

type Controller interface {
	Create(http.ResponseWriter, *http.Request)
	Show(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

type GetModel func() surf.Worker
