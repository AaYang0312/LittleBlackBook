package storage

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"time"
)

type Storage interface {
	Upload(ctx context.Context, reader io.Reader, size int64, objectName, contentType string) (string, error)
}

type minioStorage struct {
	client         *minio.Client
	bucket         string
	publicEndpoint string
}

func NewMinIO(endpoint, publicEndpoint, ak, sk, bucket string, useSSL bool) (Storage, error) {
	if publicEndpoint == "" {
		publicEndpoint = endpoint
	}
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(ak, sk, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := cli.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := cli.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	// demo 直接开公共读，README 演进项：预签名 URL
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, bucket)
	if err := cli.SetBucketPolicy(ctx, bucket, policy); err != nil {
		return nil, err
	}
	return &minioStorage{client: cli, bucket: bucket, publicEndpoint: publicEndpoint}, nil
}

func (m *minioStorage) Upload(ctx context.Context, reader io.Reader, size int64, objectName, contentType string) (string, error) {
	if _, err := m.client.PutObject(ctx, m.bucket, objectName, reader, size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s/%s/%s", m.publicEndpoint, m.bucket, objectName), nil
}
