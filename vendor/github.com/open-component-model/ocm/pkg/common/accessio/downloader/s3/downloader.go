// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const defaultRegion = "us-west-1"

// Downloader is a downloader capable of downloading S3 Objects.
type Downloader struct {
	region, bucket, key, version string
	creds                        *AWSCreds
}

func NewDownloader(region, bucket, key, version string, creds *AWSCreds) *Downloader {
	return &Downloader{
		region:  region,
		bucket:  bucket,
		key:     key,
		version: version,
		creds:   creds,
	}
}

// AWSCreds groups AWS related credential values together.
type AWSCreds struct {
	AccessKeyID  string
	AccessSecret string
	SessionToken string
}

func (s *Downloader) Download(w io.WriterAt) error {
	ctx := context.Background()
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(s.region),
	}
	var awsCred aws.CredentialsProvider = aws.AnonymousCredentials{}
	if s.creds != nil {
		awsCred = awscreds.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     s.creds.AccessKeyID,
				SecretAccessKey: s.creds.AccessSecret,
			},
		}
	}
	opts = append(opts, config.WithCredentialsProvider(awsCred))
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load configuration for AWS: %w", err)
	}

	if s.region == "" {
		var err error
		// deliberately use a different client so the real one will use the right region.
		// Region has to be provided to get the region of the specified bucket. We use the
		// global "default" of us-west-1 here. This will be updated to the right region
		// once we retrieve it or die trying.
		cfg.Region = defaultRegion
		s.region, err = manager.GetBucketRegion(ctx, s3.NewFromConfig(cfg), s.bucket, func(o *s3.Options) {
			o.Region = defaultRegion
		})
		if err != nil {
			return fmt.Errorf("failed to find bucket region: %w", err)
		}
		cfg.Region = s.region
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// Pass in creds because of https://github.com/aws/aws-sdk-go-v2/issues/1797
		o.Credentials = awsCred
		o.Region = s.region
	})
	downloader := manager.NewDownloader(client)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key),
	}
	if s.version != "" {
		input.VersionId = aws.String(s.version)
	}
	if _, err := downloader.Download(ctx, w, input); err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}
