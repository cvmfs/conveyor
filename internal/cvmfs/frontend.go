package cvmfs

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// startFrontEnd initializes the HTTP frontend of the job server
func startFrontEnd(cfg *Config, backend *serverBackend, keys *Keys) error {
	endpoints := cfg.HTTPEndpoints()

	router := mux.NewRouter()

	// Add the HMAC authorization middleware
	authz := hmacAuthorization{keys}
	router.Use(authz.Middleware)

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

	// POST a new job
	r = router.NewRoute()
	r.Path(endpoints.NewJobs(false))
	r.Methods("POST")
	r.Headers("Content-Type", "application/json")
	r.Headers("Authorization", "")
	r.HandlerFunc(makePutNewJobHandler(backend))

	// GET the status of multiple completed jobs
	r = router.NewRoute()
	r.Path(endpoints.CompletedJobs(false))
	r.Methods("GET")
	r.Queries("id", "", "full", "")
	r.Headers("Authorization", "")
	r.HandlerFunc(makeGetJobStatusHandler(backend))

	// POST the completion status of a job
	r = router.NewRoute()
	r.Path(endpoints.CompletedJobs(false))
	r.Methods("POST")
	r.Headers("Content-Type", "application/json")
	r.Headers("Authorization", "")
	r.HandlerFunc(makePutJobStatusHandler(backend))

	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		return errors.Wrap(err, "front-end server error")
	}

	return nil
}
