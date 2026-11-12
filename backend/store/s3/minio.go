package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
)

type MinioClient struct {
	*minio.Client
	config *config.Config
}

func NewMinioClient(config *config.Config) (*MinioClient, error) {
	endpoint := config.S3.Endpoint
	accessKey := config.S3.AccessKey
	secretKey := config.S3.SecretKey

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	// check bucket
	bucket := domain.Bucket
	exists, err := minioClient.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = minioClient.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{
			Region: "us-east-1",
		})
		if err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}
	// 桶保持私有，文件统一经 API 的 /static-file 鉴权读取；旧版本建桶时设置过 public-read，这里清掉
	policy, err := minioClient.GetBucketPolicy(context.Background(), bucket)
	if err != nil {
		return nil, fmt.Errorf("get bucket policy: %w", err)
	}
	if policy != "" {
		if err := minioClient.SetBucketPolicy(context.Background(), bucket, ""); err != nil {
			return nil, fmt.Errorf("remove bucket policy: %w", err)
		}
	}
	return &MinioClient{Client: minioClient, config: config}, nil
}

// sign url
func (c *MinioClient) SignURL(ctx context.Context, bucket, object string, expires time.Duration) (string, error) {
	url, err := c.PresignedGetObject(ctx, bucket, object, expires, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
