package cvmfs

import (
	"strings"
	"testing"
)

const (
	goodTestKey = "plain_text ID1 SECRET1"
	badTestKey1 = "ID2 SECRET2"
	badTestKey2 = "rubbish plain_text ID3 SECRET3"
)

func TestReadKey(t *testing.T) {
	t.Run("valid key", func(t *testing.T) {
		rdr := strings.NewReader(goodTestKey)
		id, secret, err := readKey(rdr)
		if err != nil {
			t.Errorf("could not read key")
		}
		if id != "ID1" {
			t.Errorf("invalid id: %v\n", id)
		}
		if secret != "SECRET1" {
			t.Errorf("invalid secret: %v\n", secret)
		}
	})
	t.Run("invalid key (missing field)", func(t *testing.T) {
		rdr := strings.NewReader(badTestKey1)
		id, secret, err := readKey(rdr)
		if err == nil {
			t.Errorf("invalid key was not rejected - id: %v, secret: %v\n", id, secret)
		}
	})
	t.Run("invalid key (extra content)", func(t *testing.T) {
		rdr := strings.NewReader(badTestKey2)
		id, secret, err := readKey(rdr)
		if err == nil {
			t.Errorf("invalid key was not rejected - id: %v, secret: %v\n", id, secret)
		}
	})
}
