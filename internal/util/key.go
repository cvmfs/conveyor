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

// Keys - HTTP API keys
type Keys struct {
	Secrets  map[string]string // map from keyId to secret
	RepoKeys map[string]string // map from repository name to keyId
}

func initKeys() Keys {
	return Keys{
		Secrets:  map[string]string{},
		RepoKeys: map[string]string{},
	}
}

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

	keys := initKeys()
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".gw") {
			buf, err := ioutil.ReadFile(path.Join(keyDir, f.Name()))
			if err != nil {
				return nil, errors.Wrap(
					err, fmt.Sprintf("could not read key file: %v", f))
			}
			tokens := strings.Split(string(buf), " ")
			keys.Secrets[tokens[1]] = tokens[2]
			repoName := strings.TrimSuffix(f.Name(), ".gw")
			keys.RepoKeys[repoName] = tokens[1]
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
