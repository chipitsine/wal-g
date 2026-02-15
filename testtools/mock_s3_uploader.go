package testtools

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/wal-g/wal-g/pkg/storages/memory"
)

// mockMultiFailureError simulates a multi-upload failure
type mockMultiFailureError struct {
	uploadID string
	errMsg   string
}

func (err mockMultiFailureError) Error() string {
	return err.errMsg
}

// MockS3Uploader client for S3. Compatible with SDK v2 manager.Uploader
type MockS3Uploader struct {
	multiErr bool
	err      bool
	storage  *memory.KVS
}

func NewMockS3Uploader(multiErr, err bool, storage *memory.KVS) *MockS3Uploader {
	return &MockS3Uploader{multiErr: multiErr, err: err, storage: storage}
}

// Upload simulates the manager.Uploader Upload method for SDK v2
func (uploader *MockS3Uploader) Upload(ctx context.Context, input *s3.PutObjectInput,
	opts ...func(*manager.Uploader)) (*manager.UploadOutput, error) {
	if uploader.err {
		return nil, &smithy.GenericAPIError{Code: "UploadFailed", Message: "mock Upload error"}
	}

	if uploader.multiErr {
		return nil, mockMultiFailureError{
			uploadID: "mock ID",
			errMsg:   "multiupload failure error",
		}
	}

	output := &manager.UploadOutput{
		Location:  *input.Bucket,
		VersionID: input.Key,
	}

	var err error
	if uploader.storage == nil {
		// Discard bytes to unblock pipe.
		_, err = io.Copy(io.Discard, input.Body)
	} else {
		var buf bytes.Buffer
		_, err = io.Copy(&buf, input.Body)
		uploader.storage.Store(*input.Bucket+*input.Key, buf)
	}
	if err != nil {
		return nil, err
	}

	return output, nil
}

