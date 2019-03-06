package cvmfs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// hmacAuthorization implements the Middleware interface and checks the HMAC signature of
// incoming requests
type hmacAuthorization struct {
	keys *Keys
}

func (m *hmacAuthorization) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		tokens := strings.Split(authHeader, " ")
		if len(tokens) != 2 {
			httpError(
				"Missing or incomplete Authorization header",
				&w, http.StatusUnauthorized)
			return
		}
		key := m.keys.Secrets[tokens[0]]
		HMAC, err := base64.StdEncoding.DecodeString(tokens[1])
		if err != nil {
			httpWrapError(err, "Could not base64 decode HMAC", &w, http.StatusBadRequest)
			return
		}

		buf := []byte{}
		if req.Method == "POST" {
			// For POST requests, the body of the request is used to compute the HMAC
			buf, err = ioutil.ReadAll(req.Body)
			if err != nil {
				httpWrapError(err, "reading request body failed", &w, http.StatusBadRequest)
				return
			}
			// Body needs to be read again in the next handler, reset it
			// using a copy of the original body
			bodyCopy := ioutil.NopCloser(bytes.NewReader(buf))
			req.Body.Close()
			req.Body = bodyCopy
		} else {
			// For GET requests, the query string is used to compute the HMAC
			buf = []byte(req.URL.RawQuery)
		}

		if !checkHMAC(buf, HMAC, key) {
			httpError("Invalid HMAC", &w, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func makeGetJobStatusHandler(backend *serverBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		full := false
		if req.URL.Query().Get("full") != "false" {
			full = true
		}

		ids := req.URL.Query()["id"]
		status, err := backend.getJobStatus(ids, full)
		if err != nil {
			Log.Errorln(errors.Wrap(err, "get job failed"))
		}

		rep, err := json.Marshal(status)
		if err != nil {
			httpWrapError(err, "JSON serialization failed", &w, http.StatusInternalServerError)
			return
		}

		w.Write(rep)
	}
}

func makePutNewJobHandler(backend *serverBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		buf, err := ioutil.ReadAll(req.Body)
		if err != nil {
			httpWrapError(err, "reading request body failed", &w, http.StatusBadRequest)
			return
		}

		var job JobSpecification
		if err := json.Unmarshal(buf, &job); err != nil {
			httpWrapError(err, "JSON deserialization of request failed", &w, http.StatusBadRequest)
			return
		}

		status, err := backend.putNewJob(&job)
		if err != nil {
			Log.Errorln(errors.Wrap(err, "get job failed"))
		}

		rep, err := json.Marshal(status)
		if err != nil {
			httpWrapError(
				err, "JSON serialization of reply failed", &w,
				http.StatusInternalServerError)
			return
		}

		w.Write(rep)
	}
}

func makePutJobStatusHandler(backend *serverBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		buf, err := ioutil.ReadAll(req.Body)
		if err != nil {
			httpWrapError(err, "reading request body failed", &w, http.StatusBadRequest)
			return
		}

		var job ProcessedJob
		if err := json.Unmarshal(buf, &job); err != nil {
			httpWrapError(err, "JSON deserialization of request failed", &w, http.StatusBadRequest)
			return
		}

		status, err := backend.putJobStatus(&job)
		if err != nil {
			Log.Errorln(errors.Wrap(err, "get job failed"))
		}

		rep, err := json.Marshal(status)
		if err != nil {
			httpWrapError(err, "JSON serialization of reply failed", &w, http.StatusInternalServerError)
			return
		}

		w.Write(rep)
	}
}

func httpError(msg string, w *http.ResponseWriter, code int) {
	Log.Errorln(errors.New(msg))
	http.Error(*w, msg, code)
}

func httpWrapError(err error, msg string, w *http.ResponseWriter, code int) {
	Log.Errorln(errors.Wrap(err, msg))
	http.Error(*w, msg, code)
}
