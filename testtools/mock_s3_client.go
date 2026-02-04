package testtools

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	walgs3 "github.com/wal-g/wal-g/pkg/storages/s3"
)

// Mock out S3 client. Implements manager.UploadAPIClient interface for SDK v2
type MockS3Client struct {
	err      bool
	notFound bool
}

func NewMockS3Client(err, notFound bool) *MockS3Client {
	return &MockS3Client{err: err, notFound: notFound}
}

// PutObject for SDK v2
func (client *MockS3Client) PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockPutObject", Message: "mock PutObject error"}
	}
	return &s3.PutObjectOutput{}, nil
}

// UploadPart for SDK v2 (required by manager.UploadAPIClient)
func (client *MockS3Client) UploadPart(ctx context.Context, input *s3.UploadPartInput, opts ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockUploadPart", Message: "mock UploadPart error"}
	}
	return &s3.UploadPartOutput{ETag: aws.String("mock-etag")}, nil
}

// CreateMultipartUpload for SDK v2 (required by manager.UploadAPIClient)
func (client *MockS3Client) CreateMultipartUpload(ctx context.Context, input *s3.CreateMultipartUploadInput, opts ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockCreateMultipartUpload", Message: "mock CreateMultipartUpload error"}
	}
	return &s3.CreateMultipartUploadOutput{UploadId: aws.String("mock-upload-id")}, nil
}

// CompleteMultipartUpload for SDK v2 (required by manager.UploadAPIClient)
func (client *MockS3Client) CompleteMultipartUpload(ctx context.Context, input *s3.CompleteMultipartUploadInput, opts ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockCompleteMultipartUpload", Message: "mock CompleteMultipartUpload error"}
	}
	return &s3.CompleteMultipartUploadOutput{}, nil
}

// AbortMultipartUpload for SDK v2 (required by manager.UploadAPIClient)
func (client *MockS3Client) AbortMultipartUpload(ctx context.Context, input *s3.AbortMultipartUploadInput, opts ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockAbortMultipartUpload", Message: "mock AbortMultipartUpload error"}
	}
	return &s3.AbortMultipartUploadOutput{}, nil
}

// ListObjects for SDK v2
func (client *MockS3Client) ListObjects(ctx context.Context, input *s3.ListObjectsInput, opts ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockListObjects", Message: "mock ListObjects errors"}
	}

	contents := fakeContents()
	output := &s3.ListObjectsOutput{
		Contents: contents,
		Name:     input.Bucket,
	}

	return output, nil
}

// ListObjectsV2 for SDK v2
func (client *MockS3Client) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, opts ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockListObjectsV2", Message: "mock ListObjectsV2 errors"}
	}

	contents := fakeContents()
	output := &s3.ListObjectsV2Output{
		Contents: contents,
		Name:     input.Bucket,
	}

	return output, nil
}

// DeleteObjects for SDK v2
func (client *MockS3Client) DeleteObjects(ctx context.Context, input *s3.DeleteObjectsInput, opts ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockDeleteObjects", Message: "mock DeleteObjects error"}
	}
	return &s3.DeleteObjectsOutput{}, nil
}

// ListObjectVersions for SDK v2
func (client *MockS3Client) ListObjectVersions(ctx context.Context, input *s3.ListObjectVersionsInput, opts ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockListObjectVersions", Message: "mock ListObjectVersions error"}
	}
	return &s3.ListObjectVersionsOutput{}, nil
}

// GetBucketVersioning for SDK v2
func (client *MockS3Client) GetBucketVersioning(ctx context.Context, input *s3.GetBucketVersioningInput, opts ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockGetBucketVersioning", Message: "mock GetBucketVersioning error"}
	}
	return &s3.GetBucketVersioningOutput{}, nil
}

// GetObject for SDK v2
func (client *MockS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockGetObject", Message: "mock GetObject error"}
	}

	output := &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader("mock content")),
	}

	return output, nil
}

// CopyObject for SDK v2
func (client *MockS3Client) CopyObject(ctx context.Context, input *s3.CopyObjectInput, opts ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockCopyObject", Message: "mock CopyObject error"}
	}
	return &s3.CopyObjectOutput{}, nil
}

// HeadObject for SDK v2
func (client *MockS3Client) HeadObject(ctx context.Context, input *s3.HeadObjectInput, opts ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if client.err {
		return nil, &smithy.GenericAPIError{Code: "MockHeadObject", Message: "mock HeadObject error"}
	} else if client.notFound {
		return nil, &smithy.GenericAPIError{Code: walgs3.NotFoundAWSErrorCode, Message: "mock HeadObject error"}
	}

	return &s3.HeadObjectOutput{}, nil
}

// Creates 5 fake S3 objects with Key and LastModified field for SDK v2.
func fakeContents() []types.Object {
	c := make([]types.Object, 5)

	c[0] = types.Object{
		Key:          aws.String("mockServer/base_backup/second.nop"),
		LastModified: aws.Time(time.Date(2017, 2, 2, 30, 48, 39, 651387233, time.UTC)),
	}

	c[1] = types.Object{
		Key:          aws.String("mockServer/base_backup/fourth.nop"),
		LastModified: aws.Time(time.Date(2009, 2, 27, 20, 8, 33, 651387235, time.UTC)),
	}

	c[2] = types.Object{
		Key:          aws.String("mockServer/base_backup/fifth.nop"),
		LastModified: aws.Time(time.Date(2008, 11, 20, 16, 34, 58, 651387232, time.UTC)),
	}

	c[3] = types.Object{
		Key:          aws.String("mockServer/base_backup/first.nop"),
		LastModified: aws.Time(time.Date(2020, 11, 31, 20, 3, 58, 651387237, time.UTC)),
	}

	c[4] = types.Object{
		Key:          aws.String("mockServer/base_backup/third.nop"),
		LastModified: aws.Time(time.Date(2009, 3, 13, 4, 2, 42, 651387234, time.UTC)),
	}

	return c
}

