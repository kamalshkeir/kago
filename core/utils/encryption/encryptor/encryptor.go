package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"golang.org/x/crypto/scrypt"
)

var keyEnv string

func Encrypt(data string) (string, error) {
	keyEnv = settings.Secret
	if keyEnv == "" {
		keyEnv = utils.GenerateRandomString(19)
		settings.Secret = keyEnv
	}
	keyByte, salt, err := deriveKey([]byte(keyEnv), nil)
	if err != nil {
		return "", err
	}

	blockCipher, err := aes.NewCipher([]byte(keyByte))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	ciphertext = append(ciphertext, salt...)
	return hex.EncodeToString(ciphertext), nil
}

func Decrypt(data string) (string, error) {
	if keyEnv == "" {
		return "", errors.New("no secret given, set SECRET in env file")
	}
	var salt []byte
	dataByte, _ := hex.DecodeString(data)
	if len(dataByte) > 32 {
		salt, dataByte = dataByte[len(dataByte)-32:], dataByte[:len(dataByte)-32]
	} else {
		return "", errors.New("bad token")
	}

	key, _, err := deriveKey([]byte(keyEnv), salt)
	if err != nil {
		return "", err
	}

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return "", err
	}

	nonce, ciphertext := dataByte[:gcm.NonceSize()], dataByte[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func deriveKey(password, salt []byte) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, err
		}
	}

	key, err := scrypt.Key(password, salt, 1<<10, 8, 1, 32)
	if err != nil {
		return nil, nil, err
	}

	return key, salt, nil
}
