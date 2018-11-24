package jobdb

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/auth"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
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

	rep, err := json.Marshal(status)
	if err != nil {
		msg := "JSON serialization of reply failed"
		log.Error.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Write(rep)
}

type getJobsHandler struct {
	backend *Backend
}

func (h getJobsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	full := false
	if req.URL.Query().Get("full") != "false" {
		full = true
	}

	var ids []string
	st := req.URL.Query().Get("ids")
	if st != "" {
		ids = strings.Split(st, ",")
	}

	status, err := h.backend.GetJobs(ids, full)
	if err != nil {
		log.Error.Println(errors.Wrap(err, "get job failed"))
	}

	rep, err := json.Marshal(status)
	if err != nil {
		msg := "JSON serialization failed"
		log.Error.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Write(rep)
}

type putJobHandler struct {
	backend *Backend
	keys    *auth.Keys
}

func (h putJobHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	authHeader := req.Header.Get("Authorization")
	tokens := strings.Split(authHeader, " ")
	if len(tokens) != 2 {
		msg := "Missing or incomplete Authorization header"
		log.Error.Println(msg)
		http.Error(w, msg, http.StatusUnauthorized)
		return
	}
	key := h.keys.Secrets[tokens[0]]
	HMAC, err := base64.StdEncoding.DecodeString(tokens[1])
	if err != nil {
		msg := "Could not base64 decode HMAC"
		log.Error.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		msg := "reading request body failed"
		log.Error.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if !auth.CheckHMAC(buf, HMAC, key) {
		msg := "Invalid HMAC"
		log.Error.Println(msg)
		http.Error(w, msg, http.StatusForbidden)
		return
	}

	var job job.Processed
	if err := json.Unmarshal(buf, &job); err != nil {
		msg := "JSON deserialization of request failed"
		log.Error.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	status, err := h.backend.PutJob(&job)
	if err != nil {
		log.Error.Println(errors.Wrap(err, "get job failed"))
	}

	rep, err := json.Marshal(status)
	if err != nil {
		msg := "JSON serialization of reply failed"
		log.Error.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Write(rep)
}
