package cvmfs

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
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

// LoadKeys from the given directory. The function attempts to load keys from any file
// CVMFS gateway key file (*.gw) in the given directory
func LoadKeys(keyDir string) (*Keys, error) {
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
			input, err := os.Open(path.Join(keyDir, f.Name()))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("could not open key file: %v", f))
			}
			keyID, secret, err := readKey(input)
			if err != nil {
				return nil, errors.Wrap(err, "could not read key file")
			}
			repoName := strings.TrimSuffix(f.Name(), ".gw")
			keys.RepoKeys[repoName] = keyID
			if sec, found := keys.Secrets[keyID]; found && sec != secret {
				return nil, fmt.Errorf("multiple private keys for public key id: %v", keyID)
			}
			keys.Secrets[keyID] = secret
		}
	}
	return &keys, nil
}

func readKey(reader io.Reader) (string, string, error) {
	body := make([]byte, 0)
	bufReader := bufio.NewReader(reader)
	for {
		buf := make([]byte, bufReader.Size())
		n, err := bufReader.Read(buf)
		if err != nil && err != io.EOF {
			return "", "", errors.Wrap(err, "could not read input")
		}
		if n != 0 {
			body = append(body, buf[:n]...)
		} else {
			break
		}
	}
	tokens := strings.Split(string(body), " ")
	if len(tokens) != 3 {
		return "", "", fmt.Errorf("invalid format")
	}

	return tokens[1], tokens[2], nil
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
		return "", "", fmt.Errorf("Key not found for repository: %v", repo)
	}
	secret, present := k.Secrets[id]
	if !present {
		return "", "", fmt.Errorf("Secret not found for keyID: %v", id)
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
