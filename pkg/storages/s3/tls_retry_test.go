package s3

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An image without a CA bundle fails every handshake with an x509 error.
// Retrying it only hides the cause until the retry budget is exhausted.
func TestUntrustedCertificateIsNotRetried(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(server.Close)

	var attempts atomic.Int32
	transport := &http.Transport{}
	client := awss3.NewFromConfig(aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("key", "secret", ""),
		Retryer: newRetryerFunc(&Config{
			MaxRetries:              5,
			MaxThrottlingRetryDelay: time.Nanosecond,
		}),
		HTTPClient:   roundTripCounter{transport: transport, attempts: &attempts},
		BaseEndpoint: aws.String(server.URL),
	})
	_, err := client.ListObjectsV2(t.Context(), &awss3.ListObjectsV2Input{Bucket: aws.String("bucket")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate")
	assert.EqualValues(t, 1, attempts.Load())
	assert.EqualValues(t, 0, requests.Load())
}

type roundTripCounter struct {
	transport http.RoundTripper
	attempts  *atomic.Int32
}

func (c roundTripCounter) Do(req *http.Request) (*http.Response, error) {
	c.attempts.Add(1)
	return c.transport.RoundTrip(req)
}
