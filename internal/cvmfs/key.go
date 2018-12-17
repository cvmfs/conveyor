package cvmfs

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
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

func initKeys() Keys {
	return Keys{
		Secrets:  map[string]string{},
		RepoKeys: map[string]string{},
	}
}

// getKeyForRepo returns the keyId and secret key associated with a repository
func (k *Keys) getKeyForRepo(repo string) (string, string, error) {
	id, present := k.RepoKeys[repo]
	if !present {
		return "", "", errors.New(
			fmt.Sprintf("Key not found for repository: %v", repo))
	}
	secret, present := k.Secrets[id]
	if !present {
		return "", "", errors.New(
			fmt.Sprintf("Secret not found for keyID: %v", id))
	}

	return id, secret, nil
}

// Returns the first (alphabetically) key ID
func (k *Keys) firstKeyID() string {
	ks := make([]string, len(k.Secrets))
	idx := 0
	for keyID := range k.Secrets {
		ks[idx] = keyID
		idx++
	}
	sort.Strings(ks)
	return ks[0]
}

// computeHMAC - compute the HMAC of a message using a specific key
func computeHMAC(message []byte, key string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(message)
	return mac.Sum(nil)
}

// checkHMAC - checks the HMAC of a message
func checkHMAC(message, messageHMAC []byte, key string) bool {
	return hmac.Equal(messageHMAC, computeHMAC(message, key))
}
