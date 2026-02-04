package s3_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
	walgs3 "github.com/wal-g/wal-g/pkg/storages/s3"
)

// MockS3ClientForDeleteObjects is a mock S3 client specifically for testing DeleteObjects
// It simulates the scenario where DeleteObjects fails, triggering the error logging path
// that would cause a nil pointer panic if VersionId is not checked for nil
type MockS3ClientForDeleteObjects struct {
	s3iface.S3API
	returnError       bool
	versioningEnabled bool
}

func (m *MockS3ClientForDeleteObjects) DeleteObjects(input *s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {
	if m.returnError {
		// Return an error to trigger the error logging path in DeleteObjects
		return nil, awserr.New("MockDeleteObjectsError", "simulated delete error", nil)
	}
	return &s3.DeleteObjectsOutput{}, nil
}

func (m *MockS3ClientForDeleteObjects) GetBucketVersioning(input *s3.GetBucketVersioningInput) (*s3.GetBucketVersioningOutput, error) {
	if m.versioningEnabled {
		return &s3.GetBucketVersioningOutput{
			Status: aws.String(s3.BucketVersioningStatusEnabled),
		}, nil
	}
	return &s3.GetBucketVersioningOutput{}, nil
}

func (m *MockS3ClientForDeleteObjects) ListObjectVersions(input *s3.ListObjectVersionsInput) (*s3.ListObjectVersionsOutput, error) {
	// Return versions with nil VersionId to simulate non-versioned buckets
	return &s3.ListObjectVersionsOutput{
		Versions: []*s3.ObjectVersion{
			{
				Key:       input.Prefix,
				VersionId: nil, // This is the critical case - VersionId can be nil in non-versioned buckets
			},
		},
	}, nil
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
