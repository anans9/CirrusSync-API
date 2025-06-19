package config

// S3Config holds configuration for the S3 client
type S3Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	DisableSSL      bool
	ForcePathStyle  bool
	BucketName      string
}

// LoadS3Config loads S3 configuration from environment variables
func LoadS3Config() *S3Config {
	config := &S3Config{
		Region:          getEnv("AWS_REGION", "us-east-1"),
		AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		Endpoint:        getEnv("S3_ENDPOINT", ""),
		DisableSSL:      getEnvAsBool("S3_DISABLE_SSL", false),
		ForcePathStyle:  getEnvAsBool("S3_FORCE_PATH_STYLE", false),
		BucketName:      getEnv("S3_BUCKET_NAME", "cirrussync"),
	}

	if config.Region == "" {
		config.Region = "us-east-1"
	}

	// Return nil if credentials are missing
	if config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return nil
	}

	return config
}
