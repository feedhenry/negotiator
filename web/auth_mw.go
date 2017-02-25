package web

import (
	"net/http"

	"github.com/feedhenry/negotiator/pkg/log"
)

const authHeader = "X-AUTH"

// Auth handles auth
type Auth struct {
	authKey string
	logger  log.Logger
}

// Auth handles authenticating requests
func (a Auth) Auth(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	a.logger.Info("request recieved")
	var auth = req.Header.Get(authHeader)
	if a.authKey == "" {
		a.logger.Info("authentication is disabled. Set an authKey to enable it ")
		next(w, req)
		return
	}
	if a.authKey != auth {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	next(w, req)
}
