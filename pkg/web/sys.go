package web

import (
	"encoding/json"
	"net/http"
)

// SysHandler is the handler for the /sys route
type SysHandler struct{}

// Ping is a simple liveliness check
func (s SysHandler) Ping(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-type", "text/plain")
	res.Write([]byte("Ok"))
}

// Health checks dependencies and responds if the sevice feels it is healthy
func (s SysHandler) Health(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-type", "application/json")
	health := map[string]string{}
	encoder := json.NewEncoder(res)
	if err := encoder.Encode(health); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
	}
}
