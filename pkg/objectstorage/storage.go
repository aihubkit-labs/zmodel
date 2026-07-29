package objectstorage

import (
	"context"
	"io"
	"time"
)

type Config struct {
	Endpoint        string
	Region          string
	AccessKey       string
	SecretAccessKey string
}

type PutObjectInput struct {
	Bucket      string
	Key         string
	Body        io.Reader
	ContentType string
	Metadata    map[string]string
}

type PutObjectResult struct {
	ETag string
}

type HeadObjectInput struct {
	Bucket string
	Key    string
}

type HeadObjectResult struct {
	Exists        bool
	ContentType   string
	ContentLength int64
	ETag          string
	LastModified  time.Time
	Metadata      map[string]string
}

type DeleteObjectInput struct {
	Bucket string
	Key    string
}

type PresignGetObjectInput struct {
	Bucket              string
	Key                 string
	Expires             time.Duration
	ResponseContentType string
	ResponseDisposition string
}

type Storage interface {
	PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error)
	HeadObject(ctx context.Context, input HeadObjectInput) (HeadObjectResult, error)
	DeleteObject(ctx context.Context, input DeleteObjectInput) error
	PresignGetObject(ctx context.Context, input PresignGetObjectInput) (string, error)
}

type Factory func(ctx context.Context, cfg Config) (Storage, error)

var NewStorage Factory = func(ctx context.Context, cfg Config) (Storage, error) {
	return NewS3Storage(ctx, cfg)
}
