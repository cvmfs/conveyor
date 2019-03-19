package cvmfs

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	downloadTimeout = 30 // max download timeout in seconds
)

func downloadFile(destDir, src string, timeoutSec int) error {
	srcURL, err := url.Parse(src)
	if err != nil {
		return errors.Wrap(err, "could not parse source URL")
	}
	fileName := srcURL.Path
	targetFile := path.Join(destDir, fileName)

	checksum := srcURL.Query().Get("checksum")
	hasChecksum := len(checksum) > 0
	digest := make([]byte, 0)
	var algorithm string
	if hasChecksum {
		var err error
		digest, algorithm, err = parseChecksum(checksum)
		if err != nil {
			return errors.Wrap(
				err, fmt.Sprintf("invalid checksum query parameter %v\n", checksum))
		}

		// If the target file exists and has the correct digest, skip download
		if checkDigest(targetFile, digest, algorithm) == nil {
			return nil
		}
	}

	client := http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}
	rep, err := client.Get(src)
	if err != nil {
		return errors.Wrap(err, "could not make GET request")
	}
	defer rep.Body.Close()

	fout, err := os.OpenFile(targetFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return errors.Wrap(err, "could not create destination file")
	}
	defer fout.Close()

	if _, err := io.Copy(fout, rep.Body); err != nil {
		return errors.Wrap(err, "could not read reply body")
	}

	if hasChecksum {
		if err := checkDigest(targetFile, digest, algorithm); err != nil {
			return errors.Wrap(
				err, "destination file failed integrity check")
		}
	}

	return nil
}

func parseChecksum(checksum string) ([]byte, string, error) {
	digest := make([]byte, 0)
	var algorithm string
	tokens := strings.Split(checksum, ":")
	if len(tokens) > 1 {
		algorithm = tokens[0]
		var err error
		digest, err = hex.DecodeString(tokens[1])
		if err != nil {
			errors.Wrap(err, "could not decode digest")
		}
	} else {
		var err error
		digest, err = hex.DecodeString(tokens[0])
		if err != nil {
			errors.Wrap(err, "could not decode digest")
		}
		switch len(digest) {
		case md5.Size:
			algorithm = "md5"
		case sha1.Size:
			algorithm = "sha1"
		case sha256.Size:
			algorithm = "sha256"
		default:
			return []byte{}, "", errors.New("invalid checksum")
		}
	}
	return digest, algorithm, nil
}

func checkDigest(fileName string, digest []byte, algorithm string) error {
	fin, err := os.Open(fileName)
	if err != nil {
		return errors.Wrap(
			err, "could not open target file for reading")
	}
	defer fin.Close()
	newDigest, err := computeHash(algorithm, fin)
	if err != nil {
		return errors.Wrap(
			err, "could not compute hash of target file")
	}

	if !bytes.Equal(newDigest, digest) {
		return errors.New(
			fmt.Sprintf("hash mismatch - expected: %v, found: %v\n",
				digest, newDigest))
	}

	return nil
}

func computeHash(algorithm string, in io.Reader) ([]byte, error) {
	var h hash.Hash
	switch algorithm {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	default:
		return []byte{}, errors.New("invalid hash algorithm")
	}

	if _, err := io.Copy(h, in); err != nil {
		return []byte{}, errors.Wrap(err, "could not compute hash")
	}

	return h.Sum(nil), nil
}
