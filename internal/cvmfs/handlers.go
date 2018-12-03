package cvmfs

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type getJobsHandler struct {
	backend *Backend
}

func (h getJobsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	full := false
	if req.URL.Query().Get("full") != "false" {
		full = true
	}

	ids := req.URL.Query()["id"]
	status, err := h.backend.getJobs(ids, full)
	if err != nil {
		LogError.Println(errors.Wrap(err, "get job failed"))
	}

	rep, err := json.Marshal(status)
	if err != nil {
		msg := "JSON serialization failed"
		LogError.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Write(rep)
}

type putJobHandler struct {
	backend *Backend
	keys    *Keys
}

func (h putJobHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	authHeader := req.Header.Get("Authorization")
	tokens := strings.Split(authHeader, " ")
	if len(tokens) != 2 {
		msg := "Missing or incomplete Authorization header"
		LogError.Println(msg)
		http.Error(w, msg, http.StatusUnauthorized)
		return
	}
	key := h.keys.Secrets[tokens[0]]
	HMAC, err := base64.StdEncoding.DecodeString(tokens[1])
	if err != nil {
		msg := "Could not base64 decode HMAC"
		LogError.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		msg := "reading request body failed"
		LogError.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if !CheckHMAC(buf, HMAC, key) {
		msg := "Invalid HMAC"
		LogError.Println(msg)
		http.Error(w, msg, http.StatusForbidden)
		return
	}

	var job ProcessedJob
	if err := json.Unmarshal(buf, &job); err != nil {
		msg := "JSON deserialization of request failed"
		LogError.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	status, err := h.backend.putJob(&job)
	if err != nil {
		LogError.Println(errors.Wrap(err, "get job failed"))
	}

	rep, err := json.Marshal(status)
	if err != nil {
		msg := "JSON serialization of reply failed"
		LogError.Println(errors.Wrap(err, msg))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Write(rep)
}
