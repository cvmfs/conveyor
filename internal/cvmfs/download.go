package cvmfs

import (
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

	checksum := srcURL.Query().Get("checksum")
	digest, checksumType, err := parseChecksum(checksum)
	if err != nil {
		return errors.Wrap(
			err, fmt.Sprintf("invalid checksum query parameter %v\n", checksum))
	}

	targetFile := path.Join(destDir, fileName)

	// If the target file exists and has the correct digest, skip download
	if digest != "" && checkDigest(targetFile, digest, checksumType) == nil {
		return nil
	}

	client := http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}
	rep, err := client.Get(src)
	if err != nil {
		return errors.Wrap(err, "could not make GET request")
	}
	defer rep.Body.Close()

	fout, err := os.OpenFile(targetFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		errors.Wrap(err, "could not create destination file")
	}
	defer fout.Close()

	if _, err := io.Copy(fout, rep.Body); err != nil {
		errors.Wrap(err, "could not read reply body")
	}

	if digest != "" {
		if err := checkDigest(targetFile, digest, checksumType); err != nil {
			return errors.Wrap(
				err, "destination file failed integrity check")
		}
	}

	return nil
}

func parseChecksum(checksum string) (string, string, error) {
	var digest string
	var checksumType string
	if checksum != "" {
		tokens := strings.Split(checksum, ":")
		if len(tokens) > 1 {
			checksumType = tokens[0]
			digest = tokens[1]
		} else {
			digest = tokens[0]
			switch len(digest) {
			case 32:
				checksumType = "md5"
			case 40:
				checksumType = "sha1"
			case 64:
				checksumType = "sha256"
			default:
				return "", "", errors.New("invalid checksum")
			}
		}
	}
	return digest, checksumType, nil
}

func checkDigest(fileName, digest, checksumType string) error {
	fin, err := os.Open(fileName)
	if err != nil {
		return errors.Wrap(
			err, "could not open target file for reading")
	}
	defer fin.Close()
	newDigest, err := computeHash(checksumType, fin)
	if err != nil {
		return errors.Wrap(
			err, "could not compute hash of target file")
	}

	if newDigest != digest {
		return errors.New(
			fmt.Sprintf("hash mismatch - expected: %v, found: %v\n",
				digest, newDigest))
	}

	return nil
}

func computeHash(algorithm string, in io.Reader) (string, error) {
	var h hash.Hash
	switch algorithm {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	default:
		return "", errors.New("invalid hash algorithm")
	}

	if _, err := io.Copy(h, in); err != nil {
		return "", errors.Wrap(err, "could not compute hash")
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
