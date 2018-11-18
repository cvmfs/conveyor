package jobdb

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

var apiRoot = "/api/v1"

func getHome(w http.ResponseWriter, h *http.Request) {
	r := fmt.Sprintf(
		"You are in an open field on the west side" +
			"of a white house with a boarded front door.\n")
	w.Write([]byte(r))
}

func getJob(w http.ResponseWriter, h *http.Request) {
	ids := mux.Vars(h)["id"]
	full := h.URL.Query()["full"]
	r := fmt.Sprintln("get job:", ids, " full:", full)
	w.Write([]byte(r))
}

func getJobs(w http.ResponseWriter, h *http.Request) {
	r := fmt.Sprintln("get jobs:")
	w.Write([]byte(r))
}

func startFrontEnd(port int) error {
	router := mux.NewRouter()

	var r *mux.Route
	r = router.NewRoute()
	r.Path(apiRoot + "/")
	r.HandlerFunc(getHome)

	r = router.NewRoute()
	r.Path(apiRoot + "/jobs/{id}")
	r.Methods("GET")
	r.Queries("full", "")
	r.HandlerFunc(getJob)

	r = router.NewRoute()
	r.Path(apiRoot + "/jobs")
	r.Methods("GET")
	r.HandlerFunc(getJobs)

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
