package web

import (
	"net/http"

	"github.com/satori/go.uuid"
)

const requestIDHeader = "X-REQUEST_ID"

// CorrellationID handles setting a requestid on the request and response
func CorrellationID(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	var id = req.Header.Get(requestIDHeader)
	if id == "" {
		id = uuid.NewV4().String()
	}
	req.Header.Add(requestIDHeader, id)
	w.Header().Add(requestIDHeader, id)
	next(w, req)
}
