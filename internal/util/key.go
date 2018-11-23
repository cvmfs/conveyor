package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
)

// Keys - map from ID to Secret defining a shared key
type Keys map[string]string

// ReadKeys - read HTTP API keys from a list of files
func ReadKeys(keyFiles []string) (Keys, error) {
	keys := make(map[string]string)
	for _, f := range keyFiles {
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			return Keys{}, errors.Wrap(
				err, fmt.Sprintf("could not read key file: %v", f))
		}
		tokens := strings.Split(string(buf), " ")
		keys[tokens[1]] = tokens[2]
	}
	return Keys{}, nil
}

// CheckHMAC - checks the HMAC of a message
func CheckHMAC(message, messageHMAC []byte, key string) bool {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageHMAC, expectedMAC)
}
