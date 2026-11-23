package utils

import (
	"bytes"
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

// MatchImageType 用固定魔数识别图片类型。
// 不使用 http.DetectContentType 等 MIME 探测算法，避免 Content-Type 被嗅探结果污染。
func MatchImageType(head []byte) (contentType string, ok bool) {
	if len(head) < 3 {
		return "", false
	}
	switch {
	case bytes.HasPrefix(head, []byte{0xff, 0xd8, 0xff}):
		return "image/jpeg", true
	case bytes.HasPrefix(head, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}):
		return "image/png", true
	case bytes.HasPrefix(head, []byte("GIF87a")) || bytes.HasPrefix(head, []byte("GIF89a")):
		return "image/gif", true
	case len(head) >= 2 && head[0] == 'B' && head[1] == 'M':
		return "image/bmp", true
	case len(head) >= 12 && bytes.Equal(head[0:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return "image/webp", true
	case len(head) >= 4 && head[0] == 0x00 && head[1] == 0x00 && head[2] == 0x01 && head[3] == 0x00:
		return "image/x-icon", true
	case bytes.HasPrefix(head, []byte("II*\x00")) || bytes.HasPrefix(head, []byte("MM\x00*")):
		return "image/tiff", true
	default:
		return "", false
	}
}

// IsImageContent 校验扩展名白名单 + 文件头魔数与扩展名一致（防 MIME / 扩展名伪装）。
func IsImageContent(filename string, head []byte) bool {
	if len(head) == 0 || !IsImageFile(filename) {
		return false
	}
	ct, ok := MatchImageType(head)
	if !ok {
		return false
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	return imageExtMatchesType(ext, ct)
}

func imageExtMatchesType(ext, contentType string) bool {
	switch ext {
	case "jpg", "jpeg":
		return contentType == "image/jpeg"
	case "png":
		return contentType == "image/png"
	case "gif":
		return contentType == "image/gif"
	case "bmp":
		return contentType == "image/bmp"
	case "webp":
		return contentType == "image/webp"
	case "ico":
		return contentType == "image/x-icon"
	case "tif", "tiff":
		return contentType == "image/tiff"
	default:
		return false
	}
}

// LooksLikeMarkup 用固定前缀判断是否像 HTML/XML 标记内容（不做 MIME 嗅探）。
func LooksLikeMarkup(head []byte) bool {
	s := strings.TrimLeft(string(head), " \t\r\n")
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "<!doctype html"),
		strings.HasPrefix(lower, "<html"),
		strings.HasPrefix(lower, "<head"),
		strings.HasPrefix(lower, "<body"),
		strings.HasPrefix(lower, "<script"),
		strings.HasPrefix(lower, "<svg"),
		strings.HasPrefix(lower, "<?xml"):
		return true
	default:
		return false
	}
}

// ContentTypeByExt 仅按扩展名映射 Content-Type；危险类型回落为 octet-stream。
// Content-Type 不得来自 MIME 探测算法或客户端声明。
func ContentTypeByExt(ext string) string {
	ext = strings.ToLower(ext)
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".pdf":
		return "application/pdf"
	case ".txt", ".md", ".markdown", ".csv", ".json", ".xml", ".yaml", ".yml":
		return "text/plain; charset=utf-8"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".zip":
		return "application/zip"
	case ".epub":
		return "application/epub+zip"
	default:
		return "application/octet-stream"
	}
}
