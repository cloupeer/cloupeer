package options

import (
	"github.com/spf13/pflag"
)

var _ IOptions = (*S3Options)(nil)

type S3Options struct {
	Endpoint        string `json:"endpoint" mapstructure:"endpoint"`
	AccessKeyID     string `json:"access-key-id" mapstructure:"access-key-id"`
	SecretAccessKey string `json:"secret-access-key" mapstructure:"secret-access-key"`
	UseSSL          bool   `json:"use-ssl" mapstructure:"use-ssl"`
	BucketName      string `json:"bucket-name" mapstructure:"bucket-name"`
	Region          string `json:"region" mapstructure:"region"`
}

func NewS3Options() *S3Options {
	return &S3Options{
		Endpoint:        "s3.cloupeer.io",
		AccessKeyID:     "admin",
		SecretAccessKey: "public_cloupeer",
		UseSSL:          true,
		BucketName:      "firmware",
		Region:          "us-east-1",
	}
}

func (o *S3Options) Validate() []error {
	errors := []error{}

	// some validate

	return errors
}

func (o *S3Options) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Endpoint, "s3.endpoint", o.Endpoint, "S3 service endpoint (e.g. s3.amazonaws.com or minio.local)")
	fs.StringVar(&o.AccessKeyID, "s3.access-key-id", o.AccessKeyID, "S3 access key ID")
	fs.StringVar(&o.SecretAccessKey, "s3.secret-access-key", o.SecretAccessKey, "S3 secret access key")
	fs.BoolVar(&o.UseSSL, "s3.use-ssl", o.UseSSL, "Enable SSL for S3 connection")
	fs.StringVar(&o.BucketName, "s3.bucket-name", o.BucketName, "S3 bucket name for firmware storage")
	fs.StringVar(&o.Region, "s3.region", o.Region, "S3 region")
}
