package usecase

import (
	"testing"

	"github.com/chaitin/panda-wiki/domain"
)

var (
	pngHead  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpegHead = []byte("\xff\xd8\xff\xe0\x00\x10JFIF")
	htmlHead = []byte("<!DOCTYPE html><html><script>alert(1)</script>")
	pdfHead  = []byte("%PDF-1.7\n")
	zipHead  = []byte("PK\x03\x04\x14\x00")
	tiffHead = []byte("II*\x00\x08\x00\x00\x00")
)

func TestInspectContent(t *testing.T) {
	none := &domain.UploadDeniedExtensionsSetting{}
	cases := []struct {
		name     string
		setting  *domain.UploadDeniedExtensionsSetting
		filename string
		head     []byte
		wantErr  bool
	}{
		{"png ok", none, "a.png", pngHead, false},
		{"tiff ok", none, "a.TIFF", tiffHead, false},
		{"docx ok", none, "a.docx", zipHead, false},
		{"pdf ok", none, "a.pdf", pdfHead, false},
		{"markdown with html ok", none, "a.md", htmlHead, false},
		{"extensionless image ok", none, "image/abc", jpegHead, false},
		{"html denied", none, "a.html", htmlHead, true},
		{"svg denied", none, "a.svg", []byte("<svg xmlns='http://www.w3.org/2000/svg'></svg>"), true},
		{"unknown extension denied", none, "a.dwg", zipHead, true},
		{"html disguised as png", none, "a.png", htmlHead, true},
		{"html disguised as pdf", none, "a.pdf", htmlHead, true},
		{"extensionless non-image denied", none, "payload", htmlHead, true},
		{"setting allows extra", &domain.UploadDeniedExtensionsSetting{AllowedExtensions: []string{".dwg"}}, "a.dwg", zipHead, false},
		{"setting denies builtin allowed", &domain.UploadDeniedExtensionsSetting{DeniedExtensions: []string{"zip"}}, "a.zip", zipHead, true},
		{"setting cannot allow builtin denied", &domain.UploadDeniedExtensionsSetting{AllowedExtensions: []string{"html"}}, "a.html", htmlHead, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := inspectContent(tc.setting, tc.filename, tc.head)
			if (err != nil) != tc.wantErr {
				t.Fatalf("inspectContent(%q) err = %v, wantErr %v", tc.filename, err, tc.wantErr)
			}
		})
	}
}

func TestInspectContentStoredType(t *testing.T) {
	got, err := inspectContent(&domain.UploadDeniedExtensionsSetting{}, "image/abc", jpegHead)
	if err != nil || got != "image/jpeg" {
		t.Fatalf("extensionless jpeg content type = %q, %v", got, err)
	}
}
