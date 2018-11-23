package jobdb

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/util"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func startFrontEnd(port int, backend *Backend, keys util.Keys) error {
	router := mux.NewRouter()

	var r *mux.Route
	r = router.NewRoute()
	r.Path("/")
	r.HandlerFunc(
		func(w http.ResponseWriter, h *http.Request) {
			r := fmt.Sprintf(
				"You are in an open field on the west side " +
					"of a white house with a boarded front door.\n")
			w.Write([]byte(r))
		})

	// GET the status of a single job
	r = router.NewRoute()
	r.Path("/jobs/{id}")
	r.Methods("GET")
	r.Queries("full", "")
	r.Handler(getJobHandler{backend})

	// GET the status of multiple jobs
	r = router.NewRoute()
	r.Path("/jobs")
	r.Methods("GET")
	r.Queries("ids", "", "full", "")
	r.Handler(getJobsHandler{backend})

	// PUT the status of a job
	r = router.NewRoute()
	r.Path("/jobs")
	r.Methods("POST")
	r.Headers("Content-Type", "application/json")
	r.Headers("Authorization", "")
	r.Handler(putJobHandler{backend, keys})

	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		return errors.Wrap(err, "front-end server error")
	}

	return nil
}
