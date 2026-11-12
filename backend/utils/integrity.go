package utils

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"github.com/tjfoc/gmsm/sm3"
)

// AuditGenesisHash 哈希链第一条记录的 prev_hash
var AuditGenesisHash = strings.Repeat("0", 64)

// DeriveKey 按用途从主密钥派生子密钥，避免同一密钥在不同场景复用
func DeriveKey(secret, purpose string) []byte {
	mac := hmac.New(sm3.New, []byte(secret))
	mac.Write([]byte("panda-wiki/" + purpose))
	return mac.Sum(nil)
}

// HMACSM3Hex 对各段加长度前缀后计算 HMAC-SM3，保证不同分段方式不会得到相同输入
func HMACSM3Hex(key []byte, parts ...string) string {
	mac := hmac.New(sm3.New, key)
	for _, p := range parts {
		mac.Write([]byte(strconv.Itoa(len(p))))
		mac.Write([]byte{':'})
		mac.Write([]byte(p))
	}
	return hex.EncodeToString(mac.Sum(nil))
}

func SM3Hex(data []byte) string {
	sum := sm3.Sm3Sum(data)
	return hex.EncodeToString(sum)
}

type StreamingSM3 struct {
	h hash.Hash
}

func NewStreamingSM3() *StreamingSM3 {
	return &StreamingSM3{h: sm3.New()}
}

func (s *StreamingSM3) Write(p []byte) (int, error) {
	return s.h.Write(p)
}

func (s *StreamingSM3) SumHex() string {
	return hex.EncodeToString(s.h.Sum(nil))
}

func HexEqual(a, b string) bool {
	return hmac.Equal([]byte(strings.ToLower(a)), []byte(strings.ToLower(b)))
}

const PasswordKDFIterations = 200000

// PasswordKey 用 PBKDF2-SM3 从口令派生 16 字节 SM4 密钥
func PasswordKey(password string, salt []byte, iterations int) ([]byte, error) {
	if iterations < 10000 {
		return nil, fmt.Errorf("kdf iterations too small: %d", iterations)
	}
	return pbkdf2.Key(sm3.New, password, salt, iterations, 16)
}

// SealSM4GCM 使用 SM4-GCM 加密，返回随机 nonce 与密文
func SealSM4GCM(key, plain []byte) (nonce, sealed []byte, err error) {
	aead, err := newSM4GCM(key)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return nonce, aead.Seal(nil, nonce, plain, nil), nil
}

var ErrSM4GCMOpen = errors.New("decrypt failed: wrong password or data corrupted")

func OpenSM4GCM(key, nonce, sealed []byte) ([]byte, error) {
	aead, err := newSM4GCM(key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, ErrSM4GCMOpen
	}
	plain, err := aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, ErrSM4GCMOpen
	}
	return plain, nil
}
