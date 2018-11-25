all: cvmfs_job

cvmfs_job:
	CGO_ENABLED=0 go build ./cmd/cvmfs_job

clean:
	rm -f cvmfs_job
