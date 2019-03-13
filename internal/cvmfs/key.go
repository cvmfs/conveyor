package cvmfs

import (
	"crypto/hmac"
	"crypto/sha256"
)

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
