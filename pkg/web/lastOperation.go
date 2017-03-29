package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/feedhenry/negotiator/pkg/deploy"
	"github.com/feedhenry/negotiator/pkg/log"
	"github.com/feedhenry/negotiator/pkg/status"
	"github.com/gorilla/mux"
)

// StatusRetriever defines how deploy status should be retrieved
type StatusRetriever interface {
	Get(key string) (*deploy.Status, error)
}

// LastOperationHandler handles returning the status of the last operation on a service. For example a provision
type LastOperationHandler struct {
	statusRetriever StatusRetriever
	logger          log.Logger
}

// NewLastOperationHandler return a snew LastOperationHandler
func NewLastOperationHandler(statusRet StatusRetriever, logger log.Logger) LastOperationHandler {
	return LastOperationHandler{
		statusRetriever: statusRet,
		logger:          logger,
	}
}

// LastOperation handle the LastOperation requests
func (lah LastOperationHandler) LastOperation(rw http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	instance := params["instance_id"]
	//planID := params["plan_id"]      // not currently used
	operation := req.URL.Query().Get("operation")
	statusKey := deploy.StatusKey(instance, operation)
	status, err := lah.statusRetriever.Get(statusKey)
	if err != nil {
		lah.handleError(err, "failed to retrieve status ", rw)
		return
	}
	encoder := json.NewEncoder(rw)
	if err := encoder.Encode(status); err != nil {
		lah.handleError(err, "failed encoding response ", rw)
		return
	}

}

func (lah LastOperationHandler) handleError(err error, msg string, rw http.ResponseWriter) {
	switch err.(type) {
	case *status.ErrStatusNotExist:
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte(msg + err.Error()))
		return
	case *json.SyntaxError, deploy.ErrInvalid:
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(msg + err.Error()))
		return
	}
	lah.logger.Error(fmt.Sprintf(" unexpected error getting last operation. context: %s \n %+v", msg, err))
	rw.WriteHeader(http.StatusInternalServerError)
	rw.Write([]byte(msg + err.Error()))
}
