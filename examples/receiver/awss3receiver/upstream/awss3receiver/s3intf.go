// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package awss3receiver // import "github.com/musaprg/otelwasm/examples/receiver/awss3receiver/upstream/awss3receiver"

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stealthrocket/net/wasip1"
)

var downloadManager *manager.Downloader //nolint:golint,unused

type ListObjectsV2Pager interface {
	HasMorePages() bool
	NextPage(context.Context, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type ListObjectsAPI interface {
	NewListObjectsV2Paginator(params *s3.ListObjectsV2Input) ListObjectsV2Pager
}

type GetObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type s3ListObjectsAPIImpl struct {
	client *s3.Client
}

func newS3Client(ctx context.Context, cfg S3DownloaderConfig) (ListObjectsAPI, GetObjectAPI, error) {
	optionsFuncs := make([]func(*config.LoadOptions) error, 0)
	if cfg.Region != "" {
		optionsFuncs = append(optionsFuncs, config.WithRegion(cfg.Region))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, optionsFuncs...)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
		return nil, nil, err
	}
	s3OptionFuncs := make([]func(options *s3.Options), 0)
	if cfg.S3ForcePathStyle {
		s3OptionFuncs = append(s3OptionFuncs, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	if cfg.Endpoint != "" {
		s3OptionFuncs = append(s3OptionFuncs, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}
	s3OptionFuncs = append(s3OptionFuncs, func(o *s3.Options) {
		buildableClient := awshttp.NewBuildableClient()
		buildableClient = buildableClient.WithTransportOptions(func(t *http.Transport) {
			t.DialContext = wasip1.DialContext
		})
		o.HTTPClient = buildableClient
	})

	client := s3.NewFromConfig(awsCfg, s3OptionFuncs...)

	return &s3ListObjectsAPIImpl{client: client}, client, nil
}

func (api *s3ListObjectsAPIImpl) NewListObjectsV2Paginator(params *s3.ListObjectsV2Input) ListObjectsV2Pager {
	return s3.NewListObjectsV2Paginator(api.client, params)
}
