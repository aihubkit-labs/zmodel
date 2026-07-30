package objectstorage

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type S3Storage struct {
	client  *s3.Client
	presign *s3.PresignClient
}

func NewS3Storage(ctx context.Context, cfg Config) (*S3Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		if strings.TrimSpace(cfg.Endpoint) != "" {
			options.BaseEndpoint = aws.String(strings.TrimRight(cfg.Endpoint, "/"))
			options.UsePathStyle = true
		}
	})
	return &S3Storage{client: client, presign: s3.NewPresignClient(client)}, nil
}

func (storage *S3Storage) PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error) {
	result, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(input.Bucket),
		Key:         aws.String(input.Key),
		Body:        input.Body,
		ContentType: aws.String(input.ContentType),
		Metadata:    input.Metadata,
	})
	if err != nil {
		return PutObjectResult{}, err
	}
	return PutObjectResult{ETag: aws.ToString(result.ETag)}, nil
}

func (storage *S3Storage) HeadObject(ctx context.Context, input HeadObjectInput) (HeadObjectResult, error) {
	result, err := storage.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "404") {
			return HeadObjectResult{Exists: false}, nil
		}
		return HeadObjectResult{}, err
	}
	return HeadObjectResult{
		Exists:        true,
		ContentType:   aws.ToString(result.ContentType),
		ContentLength: aws.ToInt64(result.ContentLength),
		ETag:          aws.ToString(result.ETag),
		LastModified:  aws.ToTime(result.LastModified),
		Metadata:      result.Metadata,
	}, nil
}

func (storage *S3Storage) DeleteObject(ctx context.Context, input DeleteObjectInput) error {
	_, err := storage.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
	})
	return err
}

func (storage *S3Storage) PresignGetObject(ctx context.Context, input PresignGetObjectInput) (string, error) {
	result, err := storage.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(input.Bucket),
		Key:                        aws.String(input.Key),
		ResponseContentType:        optionalString(input.ResponseContentType),
		ResponseContentDisposition: optionalString(input.ResponseDisposition),
	}, s3.WithPresignExpires(input.Expires))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return aws.String(value)
}

var _ Storage = (*S3Storage)(nil)
