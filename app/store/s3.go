package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3Client interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// S3 stores each exchange as an independent JSON object in an existing bucket.
type S3 struct {
	client s3Client
	bucket string
	prefix string
}

var _ Store = (*S3)(nil)

// NewS3 uses the default AWS credential and region chain, including Lambda's role.
// An empty path stores exchanges at the bucket root.
func NewS3(ctx context.Context, bucket, path string) (*S3, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME is required for the S3 store")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration for S3 store: %w", err)
	}
	prefix := strings.Trim(strings.TrimSpace(path), "/")
	if prefix != "" {
		prefix += "/"
	}
	return &S3{client: s3.NewFromConfig(cfg), bucket: bucket, prefix: prefix}, nil
}

func (s *S3) Save(ctx context.Context, e Exchange) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode exchange: %w", err)
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return fmt.Errorf("generate exchange ID: %w", err)
	}
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(s.bucket),
		Key:                  aws.String(s.prefix + hex.EncodeToString(id[:]) + ".json"),
		Body:                 bytes.NewReader(body),
		ContentType:          aws.String("application/json"),
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	})
	if err != nil {
		return fmt.Errorf("save exchange to S3: %w", err)
	}
	return nil
}

func (s *S3) List(ctx context.Context, limit int) ([]Exchange, error) {
	out := make([]Exchange, 0)
	pages := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.prefix),
	})
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list S3 exchanges: %w", err)
		}
		for _, object := range page.Contents {
			if object.Key == nil || !strings.HasSuffix(*object.Key, ".json") {
				continue
			}
			e, err := s.readExchange(ctx, *object.Key)
			if err != nil {
				return nil, err
			}
			out = append(out, e)
		}
	}
	// S3 orders by object key, not the exchange timestamp. Apply the limit only
	// after reading all pages so it selects the newest exchanges.
	slices.SortStableFunc(out, func(a, b Exchange) int {
		return b.Timestamp.Compare(a.Timestamp)
	})
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (s *S3) readExchange(ctx context.Context, key string) (Exchange, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return Exchange{}, fmt.Errorf("read S3 exchange %q: %w", key, err)
	}
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return Exchange{}, fmt.Errorf("read S3 exchange body %q: %w", key, err)
	}
	var e Exchange
	if err := json.Unmarshal(body, &e); err != nil {
		return Exchange{}, fmt.Errorf("decode S3 exchange %q: %w", key, err)
	}
	return e, nil
}
