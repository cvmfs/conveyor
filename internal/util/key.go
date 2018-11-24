package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

const (
	maxKeyFiles = 1024
)

// Keys - map from ID to Secret defining a shared key
type Keys map[string]string

// ReadKeys - read HTTP API keys from a list of files
func ReadKeys(keyDir string) (*Keys, error) {
	d, err := os.Open(keyDir)
	if err != nil {
		return nil, errors.Wrap(err, "opening key dir failed")
	}
	files, err := d.Readdir(maxKeyFiles)
	if err != nil {
		return nil, errors.Wrap(err, "key dir empty")
	}

	keys := make(map[string]string)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".gw") {
			buf, err := ioutil.ReadFile(path.Join(keyDir, f.Name()))
			if err != nil {
				return nil, errors.Wrap(
					err, fmt.Sprintf("could not read key file: %v", f))
			}
			tokens := strings.Split(string(buf), " ")
			keys[tokens[1]] = tokens[2]
		}
	}
	return &keys, nil
}

// CheckHMAC - checks the HMAC of a message
func CheckHMAC(message, messageHMAC []byte, key string) bool {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageHMAC, expectedMAC)
}
