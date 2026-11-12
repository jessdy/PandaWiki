package utils

import (
	"crypto/hmac"
	"encoding/hex"

	"github.com/tjfoc/gmsm/sm3"
)

// AnydocUploadToken 爬虫回传文件时带在上传地址里的令牌，由 JWT_SECRET 派生，api 与 consumer 计算结果一致。
func AnydocUploadToken(secret string) string {
	mac := hmac.New(sm3.New, []byte(secret))
	mac.Write([]byte("panda-wiki/anydoc-upload"))
	return hex.EncodeToString(mac.Sum(nil))
}

func ValidAnydocUploadToken(secret, token string) bool {
	if secret == "" || token == "" {
		return false
	}
	return hmac.Equal([]byte(AnydocUploadToken(secret)), []byte(token))
}
