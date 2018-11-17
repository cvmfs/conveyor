package jobdb

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/gorilla/mux"
)

func getHome(w http.ResponseWriter, h *http.Request) {
	w.Write([]byte("You are in an open field on the west side of a white house with a boarded front door."))
}

func getJob(w http.ResponseWriter, h *http.Request) {
	id := mux.Vars(h)["id"]
	log.Info.Printf("get job: %s", id)
}

func getJobs(w http.ResponseWriter, h *http.Request) {
	ids := h.URL.Query()["ids"]
	log.Info.Printf("get jobs: %s", ids)
}

func startFrontEnd(port int) error {
	router := mux.NewRouter()

	var r *mux.Route
	r = router.NewRoute()
	r.Path("/")
	r.HandlerFunc(getHome)

	r = router.NewRoute()
	r.Path("/jobs/{id}")
	r.Methods("GET")
	r.HandlerFunc(getJob)

	r = router.NewRoute()
	r.Path("/jobs")
	r.Methods("GET")
	r.Queries("ids", "")
	r.HandlerFunc(getJobs)

	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error.Println("Front-end server error:", err)
		return err
	}

	return nil
}
