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

// Mock out S3 client. Includes these methods for SDK v2:
// ListObjects(*s3.ListObjectsInput)
// ListObjectsV2(*ListObjectsV2Input)
// GetObject(*GetObjectInput)
// HeadObject(*HeadObjectInput)
type MockS3Client struct {
	err      bool
	notFound bool
}

func NewMockS3Client(err, notFound bool) *MockS3Client {
	return &MockS3Client{err: err, notFound: notFound}
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

