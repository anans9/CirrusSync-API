// pkg/s3/client.go
package s3

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"cirrussync-api/pkg/config"
)

var (
	// Global S3 client instance
	client     *Client
	clientOnce sync.Once
)

// Client wraps S3 functionality
type Client struct {
	s3Client   *s3.S3
	bucketName string
}

// NewClient initializes a new S3 client
func NewClient(config *config.S3Config) (*Client, error) {
	// Create the S3 connection
	s3Client, err := NewS3Connection(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		s3Client:   s3Client,
		bucketName: config.BucketName,
	}, nil
}

// InitS3 initializes the global S3 client instance
func InitS3(s3Config *config.S3Config) error {
	var err error
	clientOnce.Do(func() {
		client, err = NewClient(s3Config)
	})
	return err
}

// GetS3Client returns the global S3 client instance
func GetS3Client() *Client {
	return client
}

// CreateEmptyDirectory creates an empty directory marker in S3
func (c *Client) CreateEmptyDirectory(path string) error {
	// Ensure path ends with a slash
	if path[len(path)-1] != '/' {
		path = path + "/"
	}

	// Put an empty object to create the "directory"
	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(path),
		Body:   strings.NewReader(""),
	})

	return err
}

// CreateUserBaseDirectories sets up the initial directory structure for a new user
func (c *Client) CreateUserBaseDirectories(userID, volumeID string) error {
	directories := []string{
		fmt.Sprintf("users/%s/", userID),
		fmt.Sprintf("users/%s/volumes/", userID),
		fmt.Sprintf("users/%s/volumes/%s/", userID, volumeID),
		fmt.Sprintf("users/%s/volumes/%s/files/", userID, volumeID),
		fmt.Sprintf("users/%s/volumes/%s/thumbnails/", userID, volumeID),
	}

	for _, dir := range directories {
		if err := c.CreateEmptyDirectory(dir); err != nil {
			return err
		}
	}

	return nil
}

// GetUploadPresignedURL generates a presigned URL for uploading a file
func (c *Client) GetUploadPresignedURL(key string, contentType string, expiresIn time.Duration) (string, error) {
	// Create a request for the specified object
	req, _ := c.s3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(c.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})

	// Generate a presigned URL with an expiration time
	url, err := req.Presign(expiresIn)
	if err != nil {
		return "", err
	}

	return url, nil
}

// GetDownloadPresignedURL generates a presigned URL for downloading a file
func (c *Client) GetDownloadPresignedURL(key string, expiresIn time.Duration) (string, error) {
	// Create a request for the specified object
	req, _ := c.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})

	// Generate a presigned URL with an expiration time
	url, err := req.Presign(expiresIn)
	if err != nil {
		return "", err
	}

	return url, nil
}

// PrepareFileBlockUpload creates the file directory if needed and returns a presigned URL for block upload
func (c *Client) PrepareFileBlockUpload(userID, volumeID, fileID, revisionID string, blockIndex int) (string, error) {
	// Define the path for the file blocks
	fileDir := fmt.Sprintf("users/%s/volumes/%s/files/%s/", userID, volumeID, fileID)
	blockPath := fmt.Sprintf("%sblock_%s", fileDir, strconv.Itoa(blockIndex))

	// Ensure the file directory exists
	if err := c.CreateEmptyDirectory(fileDir); err != nil {
		return "", err
	}

	// Generate a presigned URL for upload, valid for 15 minutes
	uploadURL, err := c.GetUploadPresignedURL(blockPath, "application/octet-stream", 15*time.Minute)
	if err != nil {
		return "", err
	}

	return uploadURL, nil
}

// GetFileBlockDownloadURL returns a presigned URL for downloading a file block
func (c *Client) GetFileBlockDownloadURL(userID, volumeID, fileID string, blockIndex int) (string, error) {
	// Build the path to the block
	blockPath := fmt.Sprintf("users/%s/volumes/%s/files/%s/block_%s",
		userID, volumeID, fileID, strconv.Itoa(blockIndex))

	// Generate a presigned URL for download, valid for 15 minutes
	downloadURL, err := c.GetDownloadPresignedURL(blockPath, 15*time.Minute)
	if err != nil {
		return "", err
	}

	return downloadURL, nil
}

// PrepareThumbnailUpload creates thumbnail directory and returns a presigned URL for thumbnail upload
func (c *Client) PrepareThumbnailUpload(userID, volumeID, fileID, size string) (string, error) {
	// Define the path for the thumbnails
	thumbnailDir := fmt.Sprintf("users/%s/volumes/%s/thumbnails/%s/", userID, volumeID, fileID)
	thumbnailPath := fmt.Sprintf("%s%s", thumbnailDir, size) // size can be "small", "medium", "large"

	// Ensure the thumbnail directory exists
	if err := c.CreateEmptyDirectory(thumbnailDir); err != nil {
		return "", err
	}

	// Generate a presigned URL for upload, valid for 15 minutes
	uploadURL, err := c.GetUploadPresignedURL(thumbnailPath, "image/jpeg", 15*time.Minute)
	if err != nil {
		return "", err
	}

	return uploadURL, nil
}

// GetThumbnailDownloadURL returns a presigned URL for downloading a thumbnail
func (c *Client) GetThumbnailDownloadURL(userID, volumeID, fileID, size string) (string, error) {
	// Build the path to the thumbnail
	thumbnailPath := fmt.Sprintf("users/%s/volumes/%s/thumbnails/%s/%s",
		userID, volumeID, fileID, size)

	// Generate a presigned URL for download, valid for 15 minutes
	downloadURL, err := c.GetDownloadPresignedURL(thumbnailPath, 15*time.Minute)
	if err != nil {
		return "", err
	}

	return downloadURL, nil
}

// ListObjects lists objects in a directory (prefix)
func (c *Client) ListObjects(prefix string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucketName),
		Prefix: aws.String(prefix),
	}

	result, err := c.s3Client.ListObjectsV2(input)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

// DeleteObject deletes an object from S3
func (c *Client) DeleteObject(key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	}

	_, err := c.s3Client.DeleteObject(input)
	return err
}

// DeleteDirectory deletes all objects under a directory prefix
func (c *Client) DeleteDirectory(prefix string) error {
	// Ensure path ends with a slash
	if prefix[len(prefix)-1] != '/' {
		prefix = prefix + "/"
	}

	// List all objects with the prefix
	objects, err := c.ListObjects(prefix)
	if err != nil {
		return err
	}

	// If no objects were found, nothing to delete
	if len(objects) == 0 {
		return nil
	}

	// Delete each object
	for _, key := range objects {
		if err := c.DeleteObject(key); err != nil {
			return err
		}
	}

	return nil
}
