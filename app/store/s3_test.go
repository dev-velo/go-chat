package store

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type stubS3 struct {
	put  func(context.Context, *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	get  func(context.Context, *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	list func(context.Context, *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
}

func (s stubS3) PutObject(ctx context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return s.put(ctx, in)
}

func (s stubS3) GetObject(ctx context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return s.get(ctx, in)
}

func (s stubS3) ListObjectsV2(ctx context.Context, in *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return s.list(ctx, in)
}

func TestS3Save(t *testing.T) {
	want := Exchange{Prompt: "question", Response: "answer", Timestamp: time.Now().UTC()}
	keys := make(map[string]bool)
	client := stubS3{put: func(ctx context.Context, in *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
		if aws.ToString(in.Bucket) != "bucket" || !strings.HasPrefix(aws.ToString(in.Key), "exchanges/") ||
			!strings.HasSuffix(aws.ToString(in.Key), ".json") {
			t.Fatal("incorrect bucket or key")
		}
		if keys[*in.Key] {
			t.Fatal("repeated saves must not overwrite an exchange")
		}
		keys[*in.Key] = true
		if in.ServerSideEncryption != types.ServerSideEncryptionAes256 || aws.ToString(in.ContentType) != "application/json" {
			t.Fatal("expected explicit AES256 encryption and JSON content type")
		}
		var got Exchange
		if err := json.NewDecoder(in.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatal("exchange did not round trip")
		}
		return &s3.PutObjectOutput{}, nil
	}}
	store := &S3{client: client, bucket: "bucket", prefix: "exchanges/"}
	for range 2 {
		if err := store.Save(context.Background(), want); err != nil {
			t.Fatal(err)
		}
	}
}

func TestS3ListPagesAndLimits(t *testing.T) {
	base := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	for _, limit := range []int{0, 1, 10, -1} {
		calls := 0
		closed := 0
		client := stubS3{
			list: func(_ context.Context, in *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				calls++
				if aws.ToString(in.Bucket) != "bucket" || aws.ToString(in.Prefix) != "exchanges/" {
					t.Fatal("incorrect listing scope")
				}
				if calls == 1 {
					return &s3.ListObjectsV2Output{
						Contents:    []types.Object{{Key: aws.String("exchanges/a.json")}, {Key: aws.String("exchanges/")}},
						IsTruncated: aws.Bool(true), NextContinuationToken: aws.String("next"),
					}, nil
				}
				if calls != 2 || aws.ToString(in.ContinuationToken) != "next" {
					t.Fatal("expected continuation token")
				}
				return &s3.ListObjectsV2Output{Contents: []types.Object{{Key: aws.String("exchanges/b.json")}}}, nil
			},
			get: func(_ context.Context, in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				e := Exchange{Prompt: "old", Timestamp: base}
				if aws.ToString(in.Key) == "exchanges/b.json" {
					e = Exchange{Prompt: "new", Timestamp: base.Add(time.Hour)}
				}
				body, err := json.Marshal(e)
				if err != nil {
					t.Fatal(err)
				}
				return &s3.GetObjectOutput{Body: &trackedBody{Reader: strings.NewReader(string(body)), close: func() { closed++ }}}, nil
			},
		}
		store := &S3{client: client, bucket: "bucket", prefix: "exchanges/"}
		got, err := store.List(context.Background(), limit)
		if err != nil {
			t.Fatal(err)
		}
		n := 2
		if limit == 1 {
			n = 1
		}
		if len(got) != n || got[0].Prompt != "new" || calls != 2 || closed != 2 {
			t.Fatal("expected all pages read and closed, sorted newest first before limiting")
		}
	}
}

type trackedBody struct {
	io.Reader
	close func()
}

func (b *trackedBody) Close() error {
	b.close()
	return nil
}

func TestS3ListEmpty(t *testing.T) {
	store := &S3{client: stubS3{list: func(context.Context, *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
		return &s3.ListObjectsV2Output{}, nil
	}}}
	got, err := store.List(context.Background(), 0)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatal("empty history must return a non-nil empty slice")
	}
}

func TestS3Errors(t *testing.T) {
	cause := errors.New("access denied")
	client := stubS3{
		put:  func(context.Context, *s3.PutObjectInput) (*s3.PutObjectOutput, error) { return nil, cause },
		list: func(context.Context, *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) { return nil, cause },
		get:  func(context.Context, *s3.GetObjectInput) (*s3.GetObjectOutput, error) { return nil, cause },
	}
	store := &S3{client: client}
	if err := store.Save(context.Background(), Exchange{}); !errors.Is(err, cause) {
		t.Fatal("save error must propagate")
	}
	if _, err := store.List(context.Background(), 0); !errors.Is(err, cause) {
		t.Fatal("listing error must propagate")
	}
	if _, err := store.readExchange(context.Background(), "key"); !errors.Is(err, cause) {
		t.Fatal("get error must propagate")
	}
}

func TestS3RejectsCorruptJSONAndClosesBody(t *testing.T) {
	closed := false
	store := &S3{client: stubS3{get: func(context.Context, *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
		return &s3.GetObjectOutput{Body: &trackedBody{Reader: strings.NewReader("{"), close: func() { closed = true }}}, nil
	}}}
	if _, err := store.readExchange(context.Background(), "key"); err == nil || !closed {
		t.Fatal("invalid JSON must fail and close the body")
	}
}

func TestNewS3Prefix(t *testing.T) {
	for _, path := range []string{"", "exchanges", "exchanges/", "/exchanges/"} {
		store, err := NewS3(context.Background(), " bucket ", path)
		if err != nil {
			t.Fatal(err)
		}
		want := "exchanges/"
		if path == "" {
			want = ""
		}
		if store.prefix != want || store.bucket != "bucket" {
			t.Fatal("expected normalized bucket and prefix")
		}
	}
	if _, err := NewS3(context.Background(), " ", ""); err == nil {
		t.Fatal("missing bucket must fail")
	}
}
