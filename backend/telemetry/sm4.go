package telemetry

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/tjfoc/gmsm/sm3"
	"github.com/tjfoc/gmsm/sm4"
)

// Encrypt encrypts data with SM4-GCM. Key material is normalized to 16 bytes via SM3 when needed.
func Encrypt(key []byte, data []byte) (string, error) {
	if len(key) == 0 {
		return "", fmt.Errorf("empty sm4 key")
	}
	block, err := sm4.NewCipher(normalizeSM4Key(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func normalizeSM4Key(key []byte) []byte {
	if len(key) == sm4.BlockSize {
		out := make([]byte, sm4.BlockSize)
		copy(out, key)
		return out
	}
	sum := sm3.Sm3Sum(key)
	out := make([]byte, sm4.BlockSize)
	copy(out, sum[:sm4.BlockSize])
	return out
}
