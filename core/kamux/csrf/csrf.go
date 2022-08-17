package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"io"
)

const tokenLength = 64

func VerifyToken(realToken, sentToken string) bool {
	r, err := base64.StdEncoding.DecodeString(realToken)
	if err != nil {
		return false
	}
	if len(r) == 2*tokenLength {
		r = unmaskToken(r)
	}
	s, err := base64.StdEncoding.DecodeString(sentToken)
	if err != nil {
		return false
	}
	if len(s) == 2*tokenLength {
		s = unmaskToken(s)
	}
	return tokensEqual(r, s)
}

func tokensEqual(realToken, sentToken []byte) bool {
	return len(realToken) == tokenLength &&
		len(sentToken) == tokenLength &&
		subtle.ConstantTimeCompare(realToken, sentToken) == 1
}

func oneTimePad(data, key []byte) {
	n := len(data)
	if n != len(key) {
		panic("Lengths of slices are not equal")
	}

	for i := 0; i < n; i++ {
		data[i] ^= key[i]
	}
}

func MaskToken(data []byte) []byte {
	if len(data) != tokenLength {
		return nil
	}

	// tokenLength*2 == len(enckey + token)
	result := make([]byte, 2*tokenLength)
	// the first half of the result is the OTP
	// the second half is the masked token itself
	key := result[:tokenLength]
	token := result[tokenLength:]
	copy(token, data)

	// generate the random token
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}

	oneTimePad(token, key)
	return result
}
func unmaskToken(data []byte) []byte {
	if len(data) != tokenLength*2 {
		return nil
	}

	key := data[:tokenLength]
	token := data[tokenLength:]
	oneTimePad(token, key)

	return token
}
