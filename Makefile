all: cvmfs_job

cvmfs_job:
	CGO_ENABLED=0 go build ./tools/cvmfs_job

clean:
	rm -v cvmfs_job
