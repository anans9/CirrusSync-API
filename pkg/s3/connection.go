// pkg/s3/connection.go
package s3

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"cirrussync-api/pkg/config"
)

// NewS3Connection creates a new S3 client connection
func NewS3Connection(config *config.S3Config) (*s3.S3, error) {
	// Create custom AWS configuration
	awsConfig := &aws.Config{
		Region:      aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, ""),
	}

	// Configure for MinIO/custom endpoint if specified
	if config.Endpoint != "" {
		awsConfig.Endpoint = aws.String(config.Endpoint)
		awsConfig.DisableSSL = aws.Bool(config.DisableSSL)
		awsConfig.S3ForcePathStyle = aws.Bool(config.ForcePathStyle)
	}

	// Initialize AWS session
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	// Create S3 client
	client := s3.New(sess)
	return client, nil
}
