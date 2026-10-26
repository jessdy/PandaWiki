package utils

import (
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
