package utils

import (
	"bytes"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
)

func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedImageExts := []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".ico", ".tiff", ".tif",
	}

	return slices.Contains(supportedImageExts, ext)
}

// IsImageContent 用文件头校验图片，防止仅靠扩展名绕过。
func IsImageContent(filename string, head []byte) bool {
	if len(head) == 0 || !IsImageFile(filename) {
		return false
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	if ext == "tif" || ext == "tiff" {
		return bytes.HasPrefix(head, []byte("II*\x00")) || bytes.HasPrefix(head, []byte("MM\x00*"))
	}
	sniffed := http.DetectContentType(head)
	return strings.HasPrefix(sniffed, "image/")
}
