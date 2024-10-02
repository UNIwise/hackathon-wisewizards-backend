//go:generate mockgen --source=s3.go -destination=s3_mock.go -package=aws -mock_names Service=MockService
package aws

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
)

type S3Client interface {
	PutObject(
		ctx context.Context,
		bucket string,
		key string,
		payloadReader io.Reader,
	) (err error)
	DeleteObject(
		ctx context.Context,
		bucket string,
		key string,
	) (err error)
}

type S3API interface {
	PutObject(
		ctx context.Context,
		params *s3.PutObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.PutObjectOutput, error)
	DeleteObject(
		ctx context.Context,
		params *s3.DeleteObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.DeleteObjectOutput, error)
}

type S3ClientImpl struct {
	client S3API
}

func NewS3(config aws.Config) *S3ClientImpl {
	client := s3.NewFromConfig(config)

	return &S3ClientImpl{
		client: client,
	}
}

// PutObject creates an S3 object
func (c *S3ClientImpl) PutObject(ctx context.Context, bucket string, key string, payloadReader io.Reader) error {
	if _, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Body:   payloadReader,
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		return errors.Wrap(err, "Failed to put object to S3")
	}

	return nil
}

// DeleteObject deletes a single S3 object
func (c *S3ClientImpl) DeleteObject(ctx context.Context, bucket string, key string) error {
	if _, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		return errors.Wrap(err, "Failed to delete object into S3")
	}

	return nil
}
