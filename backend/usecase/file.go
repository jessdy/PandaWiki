package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/s3"
)

const (
	maxUploadBytes int64 = 100 << 20
	sniffLen             = 512
)

// 无论系统设置如何都禁止上传，可被浏览器当作页面或脚本执行。
var builtinDeniedExtensions = []string{
	"html", "htm", "svg", "svgz", "js", "mjs", "xhtml",
	"php", "jsp", "asp", "aspx", "exe", "sh", "bat", "cmd", "jar",
}

var imageExtensions = []string{"jpg", "jpeg", "png", "gif", "bmp", "webp", "ico", "tif", "tiff"}

// 白名单之外的扩展名需在系统设置 upload.allowed_extensions 中显式放行。
var builtinAllowedExtensions = slices.Concat(imageExtensions, []string{
	"mp4", "webm", "mov", "m4v", "avi", "mkv", "flv", "wmv", "3gp", "ogv",
	"mp3", "wav", "m4a", "aac", "ogg", "oga", "flac", "amr",
	"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "wps", "et", "dps", "odt", "ods", "odp", "rtf",
	"txt", "md", "markdown", "csv", "json", "xml", "yaml", "yml", "epub",
	"zip", "rar", "7z", "tar", "gz", "tgz", "bz2", "xz", "lakebook",
})

// 以纯文本或附件形式下发，内容里出现 HTML 标记也不会被渲染。
var textLikeExtensions = []string{"txt", "md", "markdown", "csv", "json", "xml", "yaml", "yml"}

type FileUsecase struct {
	logger            *log.Logger
	s3Client          *s3.MinioClient
	config            *config.Config
	systemSettingRepo *pg.SystemSettingRepo
}

func NewFileUsecase(logger *log.Logger, s3Client *s3.MinioClient, config *config.Config, systemSettingRepo *pg.SystemSettingRepo) *FileUsecase {
	return &FileUsecase{
		s3Client:          s3Client,
		logger:            logger.WithModule("usecase.file"),
		config:            config,
		systemSettingRepo: systemSettingRepo,
	}
}

func (u *FileUsecase) UploadFile(ctx context.Context, kbID string, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	head, err := readHead(src)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	contentType, err := u.inspectUpload(ctx, file.Filename, file.Size, head)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))

	filename := fmt.Sprintf("%s/%s%s", kbID, uuid.New().String(), ext)

	resp, err := u.s3Client.PutObject(
		ctx,
		domain.Bucket,
		filename,
		src,
		file.Size,
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"originalname": file.Filename,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return resp.Key, nil
}

func (u *FileUsecase) AnyDocUploadFile(ctx context.Context, file *multipart.FileHeader, path string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	objectKey, err := sanitizeObjectKey(path)
	if err != nil {
		return "", err
	}
	head, err := readHead(src)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	contentType, err := u.inspectUpload(ctx, objectKey, file.Size, head)
	if err != nil {
		return "", err
	}

	resp, err := u.s3Client.PutObject(
		ctx,
		domain.Bucket,
		objectKey,
		src,
		file.Size,
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"originalname": file.Filename,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return resp.Key, nil
}

func readHead(r io.ReadSeeker) ([]byte, error) {
	head := make([]byte, sniffLen)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return head[:n], nil
}

// inspectUpload 校验大小、扩展名白名单和文件头，返回存储时使用的 Content-Type（不采信客户端声明）。
func (u *FileUsecase) inspectUpload(ctx context.Context, filename string, size int64, head []byte) (string, error) {
	if size < 0 || size > maxUploadBytes {
		return "", fmt.Errorf("file size exceeds the %dMB limit", maxUploadBytes>>20)
	}
	setting, err := u.uploadSetting(ctx)
	if err != nil {
		return "", err
	}
	return inspectContent(setting, filename, head)
}

func inspectContent(setting *domain.UploadDeniedExtensionsSetting, filename string, head []byte) (string, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	sniffed := http.DetectContentType(head)

	if ext == "" {
		// 爬虫回传的图片可能不带扩展名，只有文件头确认是位图才放行
		if strings.HasPrefix(sniffed, "image/") {
			return sniffed, nil
		}
		return "", fmt.Errorf("files without an extension are not allowed for upload")
	}
	if !extensionAllowed(setting, ext) {
		return "", fmt.Errorf("file extension '.%s' is not allowed for upload", ext)
	}
	if slices.Contains(imageExtensions, ext) {
		if !isImageContent(ext, head, sniffed) {
			return "", fmt.Errorf("file content does not match extension '.%s'", ext)
		}
	} else if !slices.Contains(textLikeExtensions, ext) && isMarkupContent(sniffed) {
		return "", fmt.Errorf("file content does not match extension '.%s'", ext)
	}
	return safeContentType("." + ext), nil
}

func isImageContent(ext string, head []byte, sniffed string) bool {
	if ext == "tif" || ext == "tiff" {
		return bytes.HasPrefix(head, []byte("II*\x00")) || bytes.HasPrefix(head, []byte("MM\x00*"))
	}
	return strings.HasPrefix(sniffed, "image/")
}

func isMarkupContent(sniffed string) bool {
	return strings.HasPrefix(sniffed, "text/html") || strings.HasPrefix(sniffed, "text/xml")
}

func safeContentType(ext string) string {
	contentType := mime.TypeByExtension(ext)
	if contentType == "" || strings.Contains(contentType, "html") || strings.Contains(contentType, "javascript") || strings.Contains(contentType, "svg") {
		return "application/octet-stream"
	}
	return contentType
}

func sanitizeObjectKey(objectKey string) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	objectKey = strings.TrimPrefix(objectKey, "/")
	objectKey = path.Clean("/" + objectKey)
	objectKey = strings.TrimPrefix(objectKey, "/")
	if objectKey == "" || objectKey == "." || strings.Contains(objectKey, "..") || strings.Contains(objectKey, "\\") {
		return "", fmt.Errorf("invalid upload path")
	}
	return objectKey, nil
}

// extensionAllowed 先拦截内置和系统设置中的禁用扩展名，再要求命中白名单。
func extensionAllowed(setting *domain.UploadDeniedExtensionsSetting, ext string) bool {
	if slices.Contains(builtinDeniedExtensions, ext) || containsExtension(setting.DeniedExtensions, ext) {
		return false
	}
	return slices.Contains(builtinAllowedExtensions, ext) || containsExtension(setting.AllowedExtensions, ext)
}

func containsExtension(list []string, ext string) bool {
	for _, v := range list {
		if strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(v), "."), ext) {
			return true
		}
	}
	return false
}

func (u *FileUsecase) uploadSetting(ctx context.Context) (*domain.UploadDeniedExtensionsSetting, error) {
	setting, err := u.systemSettingRepo.GetSystemSetting(ctx, consts.SystemSettingUpload)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &domain.UploadDeniedExtensionsSetting{}, nil
	}
	if err != nil {
		u.logger.Error("failed to get upload extensions setting", "error", err)
		return nil, fmt.Errorf("upload check failed")
	}
	var s domain.UploadDeniedExtensionsSetting
	if len(setting.Value) > 0 {
		if err := json.Unmarshal(setting.Value, &s); err != nil {
			u.logger.Error("failed to unmarshal upload extensions setting", "error", err)
			return nil, fmt.Errorf("upload check failed")
		}
	}
	return &s, nil
}
