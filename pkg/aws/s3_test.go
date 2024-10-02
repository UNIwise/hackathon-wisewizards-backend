package aws

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

var errForTesting = errors.New("this is an error for testing")

type FakeReader struct {
	r io.Reader
}

func (fr FakeReader) Read(p []byte) (n int, err error) {
	return fr.r.Read(p)
}

type MockS3APIMethods struct {
	PutObjectFunction func(
		ctx context.Context,
		params *s3.PutObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.PutObjectOutput, error)
	DeleteObjectFunction func(
		ctx context.Context,
		params *s3.DeleteObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.DeleteObjectOutput, error)
}

func (m *MockS3APIMethods) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m.PutObjectFunction(ctx, params, optFns...)
}

func (m *MockS3APIMethods) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m.DeleteObjectFunction(ctx, params, optFns...)
}

func TestNewS3(t *testing.T) {
	s3Instance := NewS3(aws.Config{})

	assert.NotNil(t, s3Instance.client)
}

func TestPutObjectShouldFail(t *testing.T) {
	var (
		ctx    context.Context
		bucket string = "test-bucket"
		key    string = "test/key"
	)

	mockClient := &MockS3APIMethods{
		PutObjectFunction: func(
			_ctx context.Context,
			_params *s3.PutObjectInput,
			_optFns ...func(*s3.Options),
		) (*s3.PutObjectOutput, error) {
			assert.Equal(t, ctx, _ctx)
			assert.Equal(t, &bucket, _params.Bucket)
			assert.Equal(t, &key, _params.Key)
			return nil, errForTesting
		},
	}

	s3client := &S3ClientImpl{
		client: mockClient,
	}

	actualError := s3client.PutObject(ctx, bucket, key, FakeReader{})

	assert.Error(t, actualError)
}

func TestPutObjectShouldSucceed(t *testing.T) {
	var (
		ctx    context.Context
		bucket string = "test-bucket"
		key    string = "test/key"
	)

	mockClient := &MockS3APIMethods{
		PutObjectFunction: func(
			_ctx context.Context,
			_params *s3.PutObjectInput,
			_optFns ...func(*s3.Options),
		) (*s3.PutObjectOutput, error) {
			assert.Equal(t, ctx, _ctx)
			assert.Equal(t, &bucket, _params.Bucket)
			assert.Equal(t, &key, _params.Key)
			return &s3.PutObjectOutput{}, nil
		},
	}

	s3client := &S3ClientImpl{
		client: mockClient,
	}

	actualError := s3client.PutObject(ctx, bucket, key, FakeReader{})

	assert.NoError(t, actualError)
}

func TestDeleteObjectShouldFail(t *testing.T) {
	var (
		ctx    context.Context
		bucket string = "test-bucket"
		key    string = "test/key"
	)

	mockClient := &MockS3APIMethods{
		DeleteObjectFunction: func(
			_ctx context.Context,
			_params *s3.DeleteObjectInput,
			_optFns ...func(*s3.Options),
		) (*s3.DeleteObjectOutput, error) {
			assert.Equal(t, ctx, _ctx)
			assert.Equal(t, &bucket, _params.Bucket)
			assert.Equal(t, &key, _params.Key)
			return nil, errForTesting
		},
	}

	s3client := &S3ClientImpl{
		client: mockClient,
	}

	actualError := s3client.DeleteObject(ctx, bucket, key)

	assert.Error(t, actualError)
}

func TestDeleteObjectShouldSucceed(t *testing.T) {
	var (
		ctx    context.Context
		bucket string = "test-bucket"
		key    string = "test/key"
	)

	mockClient := &MockS3APIMethods{
		DeleteObjectFunction: func(
			_ctx context.Context,
			_params *s3.DeleteObjectInput,
			_optFns ...func(*s3.Options),
		) (*s3.DeleteObjectOutput, error) {
			assert.Equal(t, ctx, _ctx)
			assert.Equal(t, &bucket, _params.Bucket)
			assert.Equal(t, &key, _params.Key)
			return &s3.DeleteObjectOutput{}, nil
		},
	}

	s3client := &S3ClientImpl{
		client: mockClient,
	}

	actualError := s3client.DeleteObject(ctx, bucket, key)

	assert.NoError(t, actualError)
}
