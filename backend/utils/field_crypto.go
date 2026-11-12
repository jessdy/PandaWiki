package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/tjfoc/gmsm/sm3"
	"github.com/tjfoc/gmsm/sm4"
)

const sm4FieldPrefix = "sm4:"

var ErrFieldDecrypt = errors.New("field decrypt failed")

// FieldCipher 以 SM4-GCM 加密落库的敏感字段，密文带 "sm4:" 前缀；无前缀的值视为历史明文原样返回。
// 第一个密钥用于加密，其余密钥只用于解密轮换前写入的数据。
type FieldCipher struct {
	keys [][]byte
}

func NewFieldCipher(secrets ...string) (*FieldCipher, error) {
	fc := &FieldCipher{}
	for _, s := range secrets {
		if s == "" {
			continue
		}
		sum := sm3.Sm3Sum([]byte(s))
		fc.keys = append(fc.keys, sum[:sm4.BlockSize])
	}
	if len(fc.keys) == 0 {
		return nil, errors.New("no field encryption secret configured")
	}
	return fc, nil
}

func IsFieldEncrypted(v string) bool {
	return strings.HasPrefix(v, sm4FieldPrefix)
}

// gmsm 的 Sm4Cipher 内含共享缓冲区，不能跨 goroutine 复用，每次调用都要新建。
func newSM4GCM(key []byte) (cipher.AEAD, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (fc *FieldCipher) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	aead, err := newSM4GCM(fc.keys[0])
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(plain), nil)
	return sm4FieldPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func (fc *FieldCipher) Decrypt(value string) (string, error) {
	if !IsFieldEncrypted(value) {
		return value, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, sm4FieldPrefix))
	if err != nil {
		return "", ErrFieldDecrypt
	}
	for _, key := range fc.keys {
		aead, err := newSM4GCM(key)
		if err != nil {
			return "", err
		}
		ns := aead.NonceSize()
		if len(raw) < ns+aead.Overhead() {
			return "", ErrFieldDecrypt
		}
		if plain, err := aead.Open(nil, raw[:ns], raw[ns:], nil); err == nil {
			return string(plain), nil
		}
	}
	return "", ErrFieldDecrypt
}

// NeedsReencrypt 历史明文或旧密钥加密的值需要用当前主密钥重新加密。
func (fc *FieldCipher) NeedsReencrypt(value string) bool {
	if value == "" {
		return false
	}
	if !IsFieldEncrypted(value) {
		return true
	}
	if len(fc.keys) == 1 {
		return false
	}
	primary := &FieldCipher{keys: fc.keys[:1]}
	_, err := primary.Decrypt(value)
	return err != nil
}
