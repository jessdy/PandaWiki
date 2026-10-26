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
	"path"
	"path/filepath"
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

const maxUploadBytes int64 = 100 << 20

var builtinDeniedExtensions = []string{
	"html", "htm", "svg", "svgz", "js", "mjs", "xhtml",
	"php", "jsp", "asp", "aspx", "exe", "sh", "bat", "cmd", "jar",
}

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

func (u *FileUsecase) UploadFileGetUrl(ctx context.Context, kbID string, file *multipart.FileHeader) (string, error) {
	key, err := u.UploadFile(ctx, kbID, file)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://panda-wiki-minio:9000/static-file/%s", key), nil
}

func (u *FileUsecase) UploadFile(ctx context.Context, kbID string, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	if err := u.checkUpload(ctx, file.Filename, file.Size); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))

	filename := fmt.Sprintf("%s/%s%s", kbID, uuid.New().String(), ext)

	size := file.Size
	contentType := safeContentType(ext)

	resp, err := u.s3Client.PutObject(
		ctx,
		domain.Bucket,
		filename,
		src,
		size,
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

func (u *FileUsecase) UploadFileFromBytes(ctx context.Context, kbID string, filename string, fileBytes []byte) (string, error) {
	// Create a reader from the byte slice
	reader := bytes.NewReader(fileBytes)

	if err := u.checkUpload(ctx, filename, int64(len(fileBytes))); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(filename))

	s3Filename := fmt.Sprintf("%s/%s%s", kbID, uuid.New().String(), ext)

	size := int64(len(fileBytes))
	contentType := safeContentType(ext)

	resp, err := u.s3Client.PutObject(
		ctx,
		domain.Bucket,
		s3Filename,
		reader,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"originalname": filename,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return resp.Key, nil
}

func (u *FileUsecase) UploadFileFromReader(
	ctx context.Context,
	kbID string,
	filename string,
	reader io.Reader,
	size int64, // 必须提供对象大小
) (string, error) {
	// 生成唯一文件名
	if err := u.checkUpload(ctx, filename, size); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(filename))

	s3Filename := fmt.Sprintf("%s/%s%s", kbID, uuid.New().String(), ext)
	contentType := safeContentType(ext)

	// 上传到 S3
	_, err := u.s3Client.PutObject(
		ctx,
		domain.Bucket,
		s3Filename,
		reader,
		size, // 必须提供对象大小
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"originalname": filename,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("S3 upload failed: %w", err)
	}

	return s3Filename, nil
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
	if err := u.checkUpload(ctx, objectKey, file.Size); err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(objectKey))
	contentType := safeContentType(ext)
	size := file.Size

	resp, err := u.s3Client.PutObject(
		ctx,
		domain.Bucket,
		objectKey,
		src,
		size,
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

func (u *FileUsecase) checkUpload(ctx context.Context, filename string, size int64) error {
	if size < 0 || size > maxUploadBytes {
		return fmt.Errorf("file size exceeds the %dMB limit", maxUploadBytes>>20)
	}
	return u.checkDeniedExtension(ctx, strings.ToLower(filepath.Ext(filename)))
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

// checkDeniedExtension checks if the file extension is in the denied list
func (u *FileUsecase) checkDeniedExtension(ctx context.Context, ext string) error {
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	for _, deniedExt := range builtinDeniedExtensions {
		if ext == deniedExt {
			return fmt.Errorf("file extension '.%s' is not allowed for upload", ext)
		}
	}
	if ext == "" {
		return nil
	}

	setting, err := u.systemSettingRepo.GetSystemSetting(ctx, consts.SystemSettingUpload)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		u.logger.Error("failed to get upload denied extensions setting", "error", err)
		return fmt.Errorf("upload check failed")
	}

	var deniedSetting domain.UploadDeniedExtensionsSetting
	if err := json.Unmarshal(setting.Value, &deniedSetting); err != nil {
		u.logger.Error("failed to unmarshal denied extensions setting", "error", err)
		return fmt.Errorf("upload check failed")
	}

	for _, deniedExt := range deniedSetting.DeniedExtensions {
		if strings.EqualFold(ext, deniedExt) {
			return fmt.Errorf("file extension '.%s' is not allowed for upload", ext)
		}
	}

	return nil
}
