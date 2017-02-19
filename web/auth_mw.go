package web

import "net/http"

const authHeader = "X-AUTH"

// Auth handles auth
type Auth struct {
	authKey string
	logger  Logger
}

// Auth handles authenticating requests
func (a Auth) Auth(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
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
