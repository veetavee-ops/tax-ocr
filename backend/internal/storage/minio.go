package storage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	mc     *minio.Client
	bucket string
}

func NewClient(endpoint, accessKey, secretKey, bucket string) (*Client, error) {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	return &Client{mc: mc, bucket: bucket}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
	}
	return nil
}

type UploadResult struct {
	Path     string
	FileHash string
	Size     int64
}

func (c *Client) Upload(ctx context.Context, tenantID string, filename string, r io.Reader, size int64, contentType string) (UploadResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return UploadResult{}, err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	ext := strings.ToLower(filepath.Ext(filename))
	now := time.Now().UTC()
	objectPath := fmt.Sprintf("%s/%d/%02d/%s%s", tenantID, now.Year(), now.Month(), uuid.NewString(), ext)

	_, err = c.mc.PutObject(ctx, c.bucket, objectPath, strings.NewReader(string(data)), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return UploadResult{}, err
	}

	return UploadResult{
		Path:     objectPath,
		FileHash: hash,
		Size:     int64(len(data)),
	}, nil
}

func (c *Client) PresignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, objectPath, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) Download(ctx context.Context, objectPath string) ([]byte, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}
