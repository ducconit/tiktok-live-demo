package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Disk — driver S3-compatible (MinIO, AWS S3...) qua minio-go v7.
// Mọi thao tác ghi/đọc qua bucket — object key = tên file.
type S3Disk struct {
	name     string
	client   *minio.Client
	bucket   string
	endpoint string
	useSSL   bool
}

// NewS3Disk — kết nối S3-compatible; tự tạo bucket nếu chưa có (idempotent).
func NewS3Disk(name string, cfg config.DiskConfig) (*S3Disk, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("s3: init client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("s3: check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("s3: tạo bucket %s: %w", cfg.Bucket, err)
		}
	}

	return &S3Disk{name: name, client: client, bucket: cfg.Bucket, endpoint: cfg.Endpoint, useSSL: cfg.UseSSL}, nil
}

func (d *S3Disk) Name() string { return d.name }

func (d *S3Disk) Put(name string, content []byte) error {
	if err := validatePath(name); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := d.client.PutObject(ctx, d.bucket, name, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{
		ContentType: contentType(name),
	})
	if err != nil {
		return fmt.Errorf("s3: put %s: %w", name, err)
	}
	return nil
}

func (d *S3Disk) Get(name string) ([]byte, error) {
	if err := validatePath(name); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	obj, err := d.client.GetObject(ctx, d.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = obj.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, fmt.Errorf("s3: get %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

func (d *S3Disk) Delete(name string) error {
	if err := validatePath(name); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return d.client.RemoveObject(ctx, d.bucket, name, minio.RemoveObjectOptions{})
}

func (d *S3Disk) Exists(name string) bool {
	if err := validatePath(name); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := d.client.StatObject(ctx, d.bucket, name, minio.StatObjectOptions{})
	return err == nil
}

func (d *S3Disk) Size(name string) (int64, error) {
	if err := validatePath(name); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := d.client.StatObject(ctx, d.bucket, name, minio.StatObjectOptions{})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

// URL — URL công khai trực tiếp (bucket public read): http(s)://endpoint/bucket/key.
// Bucket private → dùng TemporaryURL (presigned).
func (d *S3Disk) URL(name string) string {
	scheme := "http"
	if d.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, d.endpoint, d.bucket, name)
}

// TemporaryURL — presigned GET (ttl giới hạn) — truy cập file bucket private.
func (d *S3Disk) TemporaryURL(name string, ttl time.Duration) (string, error) {
	if err := validatePath(name); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	u, err := d.client.PresignedGetObject(ctx, d.bucket, name, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("s3: presign %s: %w", name, err)
	}
	return u.String(), nil
}

// Health — bucket tồn tại (metrics service_up).
func (d *S3Disk) Health(ctx context.Context) error {
	_, err := d.client.BucketExists(ctx, d.bucket)
	return err
}
