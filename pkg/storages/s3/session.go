package s3

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/wal-g/tracelog"
)

func createSession(cfg *Config) (aws.Config, error) {
	ctx := context.Background()

	var optFns []func(*config.LoadOptions) error

	// TODO: Configure CA cert if provided
	// In SDK v2, custom CA bundles need to be configured through the HTTP client
	// This requires creating a custom http.Transport with the CA pool
	if cfg.CACertFile != "" {
		tracelog.WarningLogger.Printf("CA cert files are not yet fully supported in AWS SDK v2 migration: %s", cfg.CACertFile)
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("init new config: %w", err)
	}

	err = configureAWSConfig(&awsCfg, cfg)
	if err != nil {
		return aws.Config{}, fmt.Errorf("configure AWS config: %w", err)
	}

	return awsCfg, nil
}

func configureAWSConfig(awsCfg *aws.Config, cfg *Config) error {
	ctx := context.Background()

	// Configure HTTP client with logging
	// In SDK v2, we need to work with the underlying http.Client
	baseClient := awsCfg.HTTPClient
	if httpClient, ok := baseClient.(*http.Client); ok {
		if httpClient.Transport != nil {
			httpClient.Transport = NewRoundTripperWithLogging(httpClient.Transport)
		}
	}

	accessKey := cfg.AccessKey
	secretKey := cfg.Secrets.SecretKey
	sessionToken := cfg.SessionToken

	// Handle role assumption
	if cfg.RoleARN != "" {
		if os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" && os.Getenv("AWS_ROLE_ARN") != "" {
			// Skip explicit role assumption when using IRSA
			tracelog.InfoLogger.Printf("Running with IRSA, skipping explicit role assumption")
		} else {
			stsClient := sts.NewFromConfig(*awsCfg)
			assumedRole, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
				RoleArn:         aws.String(cfg.RoleARN),
				RoleSessionName: aws.String(cfg.SessionName),
			})
			if err != nil {
				return fmt.Errorf("assume role by ARN: %w", err)
			}
			accessKey = *assumedRole.Credentials.AccessKeyId
			secretKey = *assumedRole.Credentials.SecretAccessKey
			sessionToken = *assumedRole.Credentials.SessionToken
		}
	}

	// Configure credentials
	if accessKey != "" && secretKey != "" {
		staticCreds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)
		awsCfg.Credentials = staticCreds
	}

	// Configure region
	if cfg.Region != "" {
		awsCfg.Region = cfg.Region
	} else {
		region, err := detectAWSRegion(cfg.Bucket, cfg.Endpoint, awsCfg)
		if err != nil {
			return fmt.Errorf("AWS region isn't configured explicitly: detect region: %w", err)
		}
		awsCfg.Region = region
	}

	tracelog.DebugLogger.Printf("disable 100 continue %t", cfg.Disable100Continue)

	return nil
}

func detectAWSRegion(bucket, endpoint string, awsCfg *aws.Config) (string, error) {
	// If a custom endpoint is configured and it's not an AWS endpoint,
	// we're using an S3-compatible service (MinIO, Ceph, etc.)
	// In this case, use "us-east-1" as the default region
	// ref: https://github.com/minio/cookbook/blob/master/docs/aws-sdk-for-go-with-minio.md
	if endpoint != "" && !strings.HasSuffix(endpoint, ".amazonaws.com") {
		return "us-east-1", nil
	}

	// For AWS S3 or when no endpoint is specified, try to detect the region by bucket
	region, err := detectAWSRegionByBucket(bucket, awsCfg)
	if err != nil {
		return "", fmt.Errorf("detect region by bucket: %w", err)
	}
	return region, nil
}

// detectAWSRegionByBucket attempts to detect the AWS region by the bucket name
func detectAWSRegionByBucket(bucket string, cfg *aws.Config) (string, error) {
	ctx := context.Background()

	// Create a copy with temporary region set to us-east-1 for region detection
	tempCfg := cfg.Copy()
	tempCfg.Region = "us-east-1"
	s3Client := s3.NewFromConfig(tempCfg)

	output, err := s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return "", err
	}

	if output.LocationConstraint == "" {
		// buckets in "US Standard", a.k.a. us-east-1, are returned as empty region
		return "us-east-1", nil
	}
	// all other regions are strings
	return string(output.LocationConstraint), nil
}
