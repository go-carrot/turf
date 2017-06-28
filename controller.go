package turf

import (
	"net/http"
)

type Controller interface {
	Create(http.ResponseWriter, *http.Request)
	Index(w http.ResponseWriter, r *http.Request)
	Show(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}
