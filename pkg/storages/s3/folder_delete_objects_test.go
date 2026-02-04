package s3_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	walgs3 "github.com/wal-g/wal-g/pkg/storages/s3"
)

// MockS3ClientForDeleteObjects is a mock S3 client specifically for testing DeleteObjects
// It simulates the scenario where DeleteObjects fails, triggering the error logging path
// that would cause a nil pointer panic if VersionId is not checked for nil
type MockS3ClientForDeleteObjects struct {
	returnError       bool
	versioningEnabled bool
}

// Implement the minimum required methods for walgs3.API interface
func (m *MockS3ClientForDeleteObjects) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{}, nil
}

func (m *MockS3ClientForDeleteObjects) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	return &s3.CopyObjectOutput{}, nil
}

func (m *MockS3ClientForDeleteObjects) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{}, nil
}

func (m *MockS3ClientForDeleteObjects) ListObjects(ctx context.Context, params *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	return &s3.ListObjectsOutput{}, nil
}

func (m *MockS3ClientForDeleteObjects) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return &s3.ListObjectsV2Output{}, nil
}

func (m *MockS3ClientForDeleteObjects) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	if m.returnError {
		// Return an error to trigger the error logging path in DeleteObjects
		return nil, &smithy.GenericAPIError{Code: "MockDeleteObjectsError", Message: "simulated delete error"}
	}
	return &s3.DeleteObjectsOutput{}, nil
}

func (m *MockS3ClientForDeleteObjects) ListObjectVersions(
	ctx context.Context, params *s3.ListObjectVersionsInput, optFns ...func(*s3.Options),
) (*s3.ListObjectVersionsOutput, error) {
	// Return versions with nil VersionId to simulate non-versioned buckets
	return &s3.ListObjectVersionsOutput{
		Versions: []types.ObjectVersion{
			{
				Key:       params.Prefix,
				VersionId: nil, // This is the critical case - VersionId can be nil in non-versioned buckets
			},
		},
	}, nil
}

func (m *MockS3ClientForDeleteObjects) GetBucketVersioning(
	ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options),
) (*s3.GetBucketVersioningOutput, error) {
	if m.versioningEnabled {
		return &s3.GetBucketVersioningOutput{
			Status: types.BucketVersioningStatusEnabled,
		}, nil
	}
	return &s3.GetBucketVersioningOutput{}, nil
}

// TestDeleteObjects_WithNilVersionId tests that DeleteObjects handles nil VersionId gracefully
// This test validates the fix for the nil pointer panic that occurs when:
// 1. A bucket doesn't have versioning enabled
// 2. DeleteObjects fails and tries to log the objects being deleted
// 3. The error logging code tries to dereference VersionId which is nil
//
// This test is a regression test for PR #11 where the AWS SDK v2 migration
// changed VersionId from a value to a pointer, causing panics in error logging.
func TestDeleteObjects_WithNilVersionId(t *testing.T) {
	mockClient := &MockS3ClientForDeleteObjects{
		returnError:       true, // Force an error to trigger the logging path
		versioningEnabled: false,
	}

	config := &walgs3.Config{
		Bucket:           "test-bucket",
		RootPath:         "test/",
		EnableVersioning: walgs3.VersioningDisabled,
		DeleteBatchSize:  1000,
	}

	folder := walgs3.NewFolder(mockClient, nil, config.RootPath, config)

	// This should not panic even when VersionId is nil and DeleteObjects fails
	// The current code will panic on *obj.VersionId when VersionId is nil
	// After the fix, it should handle nil VersionId gracefully
	err := folder.DeleteObjects([]string{"test-object.txt"})

	// We expect an error (because our mock returns one), but we should NOT panic
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete s3 object")
}

// TestDeleteObjects_WithVersioningEnabled tests DeleteObjects with versioned bucket
// This ensures the normal versioning path works correctly
func TestDeleteObjects_WithVersioningEnabled(t *testing.T) {
	mockClient := &MockS3ClientForDeleteObjects{
		returnError:       false, // No error - successful deletion
		versioningEnabled: true,
	}

	config := &walgs3.Config{
		Bucket:           "test-bucket",
		RootPath:         "test/",
		EnableVersioning: walgs3.VersioningEnabled,
		DeleteBatchSize:  1000,
	}

	folder := walgs3.NewFolder(mockClient, nil, config.RootPath, config)

	// This should succeed without any issues
	err := folder.DeleteObjects([]string{"test-object.txt"})

	assert.NoError(t, err)
}

// TestDeleteObjects_WithVersioningDisabled tests DeleteObjects with non-versioned bucket
// This is the primary scenario where VersionId would be nil
func TestDeleteObjects_WithVersioningDisabled(t *testing.T) {
	mockClient := &MockS3ClientForDeleteObjects{
		returnError:       false, // No error - successful deletion
		versioningEnabled: false,
	}

	config := &walgs3.Config{
		Bucket:           "test-bucket",
		RootPath:         "test/",
		EnableVersioning: walgs3.VersioningDisabled,
		DeleteBatchSize:  1000,
	}

	folder := walgs3.NewFolder(mockClient, nil, config.RootPath, config)

	// This should succeed - no versions to fetch, just delete the object directly
	err := folder.DeleteObjects([]string{"test-object.txt"})

	assert.NoError(t, err)
}
