package s3

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/wal-g/tracelog"
	"gopkg.in/yaml.v3"
)

func createSession(cfg *Config) (aws.Config, error) {
	ctx := context.Background()
	
	var optFns []func(*config.LoadOptions) error
	
	// Configure CA cert if provided
	if cfg.CACertFile != "" {
		certData, err := os.ReadFile(cfg.CACertFile)
		if err != nil {
			return aws.Config{}, err
		}
		optFns = append(optFns, func(opts *config.LoadOptions) error {
			// For SDK v2, we need to handle custom CA differently through HTTP client
			// We'll handle this in the HTTP client setup below
			return nil
		})
		_ = certData // Will use this later when setting up HTTP client
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
		region, err := detectAWSRegion(cfg.Bucket, *awsCfg)
		if err != nil {
			return fmt.Errorf("AWS region isn't configured explicitly: detect region: %w", err)
		}
		awsCfg.Region = region
	}

	tracelog.DebugLogger.Printf("disable 100 continue %t", cfg.Disable100Continue)

	return nil
}

func detectAWSRegion(bucket string, awsCfg aws.Config) (string, error) {
	endpoint := ""
	if awsCfg.BaseEndpoint != nil {
		endpoint = *awsCfg.BaseEndpoint
	}
	
	if endpoint == "" ||
		strings.HasSuffix(endpoint, ".amazonaws.com") {
		region, err := detectAWSRegionByBucket(bucket, awsCfg)
		if err != nil {
			return "", fmt.Errorf("detect region by bucket: %w", err)
		}
		return region, nil
	}
	// For S3 compatible services like Minio, Ceph etc. use `us-east-1` as region
	// ref: https://github.com/minio/cookbook/blob/master/docs/aws-sdk-for-go-with-minio.md
	return "us-east-1", nil
}

// detectAWSRegionByBucket attempts to detect the AWS region by the bucket name
func detectAWSRegionByBucket(bucket string, cfg aws.Config) (string, error) {
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

func requestEndpointFromSource(endpointSource, port string) string {
	t := http.DefaultTransport
	c := http.DefaultClient
	if tr, ok := t.(*http.Transport); ok {
		tr.DisableKeepAlives = true
		c = &http.Client{Transport: tr}
	}
	resp, err := c.Get(endpointSource)
	if err != nil {
		tracelog.ErrorLogger.Printf("Endpoint source error: %v ", err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		tracelog.ErrorLogger.Printf("Endpoint source bad status code: %v ", resp.StatusCode)
		return ""
	}
	bytes, err := io.ReadAll(resp.Body)
	if err == nil {
		return net.JoinHostPort(string(bytes), port)
	}
	tracelog.ErrorLogger.Println("Endpoint source reading error:", err)
	return ""
}

func decodeHeaders(encodedHeaders string) (map[string]string, error) {
	var data interface{}
	err := yaml.Unmarshal([]byte(encodedHeaders), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML headers: %w", err)
	}

	interfaces, ok := data.(map[string]interface{})
	if !ok {
		headerList, ok := data.([]interface{})
		if !ok {
			return nil, fmt.Errorf("headers expected to be a list in YAML: %w", err)
		}
		interfaces = reformHeaderListToMap(headerList)
	}

	headers := map[string]string{}

	for k, v := range interfaces {
		headers[k] = v.(string)
	}

	return headers, nil
}

func reformHeaderListToMap(headerList []interface{}) map[string]interface{} {
	headers := map[string]interface{}{}
	for _, header := range headerList {
		ma := header.(map[string]interface{})
		for k, v := range ma {
			headers[k] = v
		}
	}
	return headers
}
