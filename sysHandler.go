package main

import "net/http"

// SysHandler is the handler for the /sys route
type SysHandler struct{}

// Ping is a simple liveliness check
func (s SysHandler) Ping(res http.ResponseWriter, req *http.Request) {
	res.Header().Add("Content-type", "text/plain")
	res.Write([]byte("Ok"))
}

// Health checks dependencies and responds if the sevice feels it is healthy
func (s SysHandler) Health(res http.ResponseWriter, req *http.Request) {

}
