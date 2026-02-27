package tigris

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/jasonchiu/envlock/core/config"
)

var ErrObjectNotFound = errors.New("object not found")

type Client struct {
	s3     *s3.Client
	bucket string
}

func NewFromProject(proj config.Project) (*Client, error) {
	accessKey := strings.TrimSpace(os.Getenv("TIGRIS_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("TIGRIS_SECRET_KEY"))
	if accessKey == "" || secretKey == "" {
		return nil, errors.New("missing Tigris credentials (set TIGRIS_ACCESS_KEY and TIGRIS_SECRET_KEY)")
	}

	endpoint := strings.TrimSpace(os.Getenv("TIGRIS_ENDPOINT"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(proj.Endpoint)
	}
	if endpoint == "" {
		return nil, errors.New("missing Tigris endpoint (set TIGRIS_ENDPOINT or project endpoint)")
	}
	region := strings.TrimSpace(os.Getenv("TIGRIS_REGION"))
	if region == "" {
		region = "auto"
	}
	if strings.TrimSpace(proj.Bucket) == "" {
		return nil, errors.New("project bucket is required")
	}

	cfg := aws.Config{
		Region: region,
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
	})
	return &Client{s3: client, bucket: proj.Bucket}, nil
}

func (c *Client) GetJSON(ctx context.Context, key string, dst any) error {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return ErrObjectNotFound
		}
		return err
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("decode %s: %w", key, err)
	}
	return nil
}

func (c *Client) PutJSON(ctx context.Context, key string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	return err
}

func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c *Client) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	p := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
	var keys []string
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Contents {
			if item.Key == nil {
				continue
			}
			keys = append(keys, *item.Key)
		}
	}
	return keys, nil
}

func isNotFound(err error) bool {
	var noSuchKey *s3types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := strings.TrimSpace(apiErr.ErrorCode())
		return code == "NoSuchKey" || code == "NotFound" || code == "NoSuchBucket"
	}
	return false
}
