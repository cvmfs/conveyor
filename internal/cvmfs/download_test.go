package cvmfs

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func serve(quit <-chan struct{}) <-chan struct{} {
	hd := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello\n")
	}
	srv := http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(hd),
	}

	go func() {
		log.Fatal(srv.ListenAndServe())
	}()

	wait := make(chan struct{})

	go func() {
		select {
		case <-quit:
			fmt.Printf("Shutting down")
			srv.Shutdown(context.Background())
			close(wait)
		}
	}()

	return wait
}

func createDummyFile(dir, content string) error {
	fname := path.Join(dir, "msg.txt")
	fout, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return errors.Wrap(err, "could not create destination file")
	}
	fout.Write([]byte(content))
	defer fout.Close()

	return nil
}

func TestMain(m *testing.M) {
	quit := make(chan struct{})
	wait := serve(quit)
	defer func() {
		close(quit)
		<-wait
	}()

	os.Exit(m.Run())
}

func TestDownload(t *testing.T) {
	// New file
	tmp, err := ioutil.TempDir("", "scratch")
	if err != nil {
		t.Fatalf("Could not create temp dir")
	}
	defer os.RemoveAll(tmp)
	testURL := "http://localhost:8080/msg.txt"
	if err := downloadFile(tmp, testURL, 10); err != nil {
		t.Errorf("Could not download file: %v", err)
	}

	// File exists with same hash
	testURL = "http://localhost:8080/msg.txt?checksum=sha1:f572d396fae9206628714fb2ce00f72e94f2258f"
	if err := downloadFile(tmp, testURL, 10); err != nil {
		t.Errorf("Could not download file: %v", err)
	}

	// File exists with same name but wrong hash
	os.RemoveAll(path.Join(tmp, "msg.txt"))
	if err := createDummyFile(tmp, "wrong\n"); err != nil {
		t.Fatalf("Could not create dummy file")
	}
	if err := downloadFile(tmp, testURL, 10); err != nil {
		t.Errorf("Could not download file: %v", err)
	}
}

func TestParseChecksum(t *testing.T) {
	t.Run("Explicit md5", func(t *testing.T) {
		digest, algorithm, err := parseChecksum(
			"md5:d77bff3a550c1bf39b78ad2136c5d604")
		if err != nil {
			t.Errorf("valid md5 checksum couldn't be parsed")
		}
		d := hex.EncodeToString(digest)
		if d != "d77bff3a550c1bf39b78ad2136c5d604" {
			t.Errorf("invalid digest was parsed: %v", d)
		}
		if algorithm != "md5" {
			t.Errorf("algorithm should be md5")
		}
	})

	t.Run("Explicit sha1", func(t *testing.T) {
		digest, algorithm, err := parseChecksum(
			"sha1:cf23df2207d99a74fbe169e3eba035e633b65d94")
		if err != nil {
			t.Errorf("valid sha1 checksum couldn't be parsed")
		}
		d := hex.EncodeToString(digest)
		if d != "cf23df2207d99a74fbe169e3eba035e633b65d94" {
			t.Errorf("invalid digest was parsed: %v", d)
		}
		if algorithm != "sha1" {
			t.Errorf("algorithm should be sha1")
		}
	})

	t.Run("Explicit sha256", func(t *testing.T) {
		digest, algorithm, err := parseChecksum(
			"sha256:1af1dfa857bf1d8814fe1af8983c18080019922e557f15a8a0d3db739d77aacb")
		if err != nil {
			t.Errorf("valid sha256 checksum couldn't be parsed")
		}
		d := hex.EncodeToString(digest)
		if d != "1af1dfa857bf1d8814fe1af8983c18080019922e557f15a8a0d3db739d77aacb" {
			t.Errorf("invalid digest was parsed: %v", d)
		}
		if algorithm != "sha256" {
			t.Errorf("algorithm should be sha256")
		}
	})

	t.Run("Detect md5", func(t *testing.T) {
		digest, algorithm, err := parseChecksum(
			"d77bff3a550c1bf39b78ad2136c5d604")
		if err != nil {
			t.Errorf("valid md5 checksum couldn't be parsed")
		}
		d := hex.EncodeToString(digest)
		if d != "d77bff3a550c1bf39b78ad2136c5d604" {
			t.Errorf("invalid digest was parsed: %v\n", d)
		}
		if algorithm != "md5" {
			t.Errorf("algorithm should be md5")
		}
	})

	t.Run("Detect sha1", func(t *testing.T) {
		digest, algorithm, err := parseChecksum(
			"cf23df2207d99a74fbe169e3eba035e633b65d94")
		if err != nil {
			t.Errorf("valid sha1 checksum couldn't be parsed")
		}
		d := hex.EncodeToString(digest)
		if d != "cf23df2207d99a74fbe169e3eba035e633b65d94" {
			t.Errorf("invalid digest was parsed: %v", d)
		}
		if algorithm != "sha1" {
			t.Errorf("algorithm should be sha1")
		}
	})

	t.Run("Detect sha256", func(t *testing.T) {
		digest, algorithm, err := parseChecksum(
			"1af1dfa857bf1d8814fe1af8983c18080019922e557f15a8a0d3db739d77aacb")
		if err != nil {
			t.Errorf("valid sha256 checksum couldn't be parsed")
		}
		d := hex.EncodeToString(digest)
		if d != "1af1dfa857bf1d8814fe1af8983c18080019922e557f15a8a0d3db739d77aacb" {
			t.Errorf("invalid digest was parsed: %v", d)
		}
		if algorithm != "sha256" {
			t.Errorf("algorithm should be sha256")
		}
	})

	t.Run("Invalid checksum", func(t *testing.T) {
		_, _, err := parseChecksum(
			"this isn't valid")
		if err.Error() != "invalid checksum" {
			t.Errorf("invalid checksum should have been rejected")
		}
	})
}

func TestComputeHash(t *testing.T) {
	t.Run("valid algorithm", func(t *testing.T) {
		rd := strings.NewReader("test input")
		digest, err := computeHash("sha1", rd)
		if err != nil {
			t.Errorf("error reading input")
		}
		d := hex.EncodeToString(digest)
		if d != "49883b34e5a0f48224dd6230f471e9dc1bdbeaf5" {
			t.Errorf("invalid digest computed: %v", d)
		}
	})
}
