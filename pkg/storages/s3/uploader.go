package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
)

type UploaderConfig struct {
	UploadConcurrency            int
	MaxPartSize                  int
	StorageClass                 string
	ServerSideEncryption         string
	ServerSideEncryptionCustomer string
	ServerSideEncryptionKMSID    string
	RetentionPeriod              int
	RetentionMode                string
}

func createUploader(s3Client *s3.Client, config *UploaderConfig) (*Uploader, error) {
	uploaderAPI := CreateUploaderAPI(s3Client, config.MaxPartSize, config.UploadConcurrency)

	if (config.ServerSideEncryption == "aws:kms") == (config.ServerSideEncryptionKMSID == "") {
		return nil, fmt.Errorf("server-side encryption KMS key ID must be set if 'aws:kms' encryption is used")
	}
	return NewUploader(
		uploaderAPI,
		config.ServerSideEncryption,
		config.ServerSideEncryptionCustomer,
		config.ServerSideEncryptionKMSID,
		config.StorageClass,
		config.RetentionMode,
		config.RetentionPeriod,
	), nil
}

type Uploader struct {
	uploaderAPI          *manager.Uploader
	serverSideEncryption string
	SSECustomerKey       string
	SSEKMSKeyID          string
	StorageClass         string
	RetentionMode        string
	RetentionPeriod      time.Duration
}

func NewUploader(uploaderAPI *manager.Uploader, serverSideEncryption, sseCustomerKey, sseKmsKeyID, storageClass,
	retentionMode string, retentionPeriod int) *Uploader {
	if retentionMode == "" {
		retentionMode = "GOVERNANCE"
	}
	return &Uploader{uploaderAPI,
		serverSideEncryption,
		sseCustomerKey,
		sseKmsKeyID,
		storageClass,
		retentionMode,
		time.Duration(retentionPeriod)}
}

func (uploader *Uploader) createUploadInput(bucket, path string, content io.Reader) *s3.PutObjectInput {
	storageClass := types.StorageClass(uploader.StorageClass)
	uploadInput := &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(path),
		Body:         content,
		StorageClass: storageClass,
	}
	if uploader.RetentionPeriod != defaultDisabledRetentionPeriod {
		mytime := time.Now().Add(time.Second * uploader.RetentionPeriod)
		mode := types.ObjectLockMode(uploader.RetentionMode)
		uploadInput.ObjectLockMode = mode
		uploadInput.ObjectLockRetainUntilDate = &mytime
	}

	if uploader.serverSideEncryption != "" {
		if uploader.SSECustomerKey != "" {
			uploadInput.SSECustomerAlgorithm = aws.String(uploader.serverSideEncryption)
			uploadInput.SSECustomerKey = aws.String(uploader.SSECustomerKey)
			customerKeyMD5 := GetSSECustomerKeyMD5(uploader.SSECustomerKey)
			uploadInput.SSECustomerKeyMD5 = aws.String(customerKeyMD5)
		} else {
			sseType := types.ServerSideEncryption(uploader.serverSideEncryption)
			uploadInput.ServerSideEncryption = sseType
		}

		if uploader.SSEKMSKeyID != "" {
			// Only aws:kms implies sseKmsKeyId, checked during validation
			uploadInput.SSEKMSKeyId = aws.String(uploader.SSEKMSKeyID)
		}
	}

	return uploadInput
}

func (uploader *Uploader) upload(ctx context.Context, bucket, path string, content io.Reader) error {
	input := uploader.createUploadInput(bucket, path, content)
	_, err := uploader.uploaderAPI.Upload(ctx, input)
	return errors.Wrapf(err, "failed to upload '%s' to bucket '%s'", path, bucket)
}

// CreateUploaderAPI returns an uploader with customizable concurrency
// and part size.
func CreateUploaderAPI(svc *s3.Client, partsize, concurrency int) *manager.Uploader {
	uploaderAPI := manager.NewUploader(svc, func(uploader *manager.Uploader) {
		uploader.PartSize = int64(partsize)
		uploader.Concurrency = concurrency
	})
	return uploaderAPI
}

func partitionStrings(strings []string, blockSize int) [][]string {
	// I've unsuccessfully tried this with interface{} but there was too much of casting
	if blockSize <= 0 {
		return [][]string{strings}
	}
	partition := make([][]string, 0)
	for i := 0; i < len(strings); i += blockSize {
		if i+blockSize > len(strings) {
			partition = append(partition, strings[i:])
		} else {
			partition = append(partition, strings[i:i+blockSize])
		}
	}
	return partition
}

func partitionObjects(objects []types.ObjectIdentifier, blockSize int) [][]types.ObjectIdentifier {
	if blockSize <= 0 {
		return [][]types.ObjectIdentifier{objects}
	}
	partition := make([][]types.ObjectIdentifier, 0)
	for i := 0; i < len(objects); i += blockSize {
		if i+blockSize > len(objects) {
			partition = append(partition, objects[i:])
		} else {
			partition = append(partition, objects[i:i+blockSize])
		}
	}
	return partition
}
