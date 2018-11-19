package jobdb

import (
	"fmt"
	"net/http"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type getJobHandler struct {
	backend *Backend
}

func (h getJobHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	full := false
	if req.URL.Query().Get("full") != "false" {
		full = true
	}

	status, err := h.backend.GetJob(id, full)
	if err != nil {
		log.Error.Println(errors.Wrap(err, "get job failed"))
	}

	rep := fmt.Sprintf("GetJob(%s, full = %v): %s\n", id, full, status)
	w.Write([]byte(rep))
}

type getJobsHandler struct {
	backend *Backend
}

func (h getJobsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rep := fmt.Sprintln("insert job:")
	w.Write([]byte(rep))
}

type putJobHandler struct {
	backend *Backend
}

func (h putJobHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rep := fmt.Sprintln("insert job:")
	w.Write([]byte(rep))
}
