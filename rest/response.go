package rest

import (
	"encoding/json"
	"fmt"
	"github.com/go-carrot/response"
	"net/http"
)

func newRestResponse(w *http.ResponseWriter, r *http.Request) *response.Response {
	return response.New().SetRenderer(
		&responseWriterRenderer{
			ResponseWriter: w,
			Request:        r,
		},
	)
}

type responseWriterRenderer struct {
	ResponseWriter *http.ResponseWriter
	Request        *http.Request
}

func (r *responseWriterRenderer) Render(resp *response.Response) string {
	// Write Header
	(*r.ResponseWriter).WriteHeader(resp.Meta.StatusCode)

	// Write Body
	b, err := json.Marshal(resp)
	if err != nil {
		panic("Unable to json.Marshal our Response")
	}
	s := string(b)
	fmt.Fprintln(*r.ResponseWriter, s)

	return s
}
