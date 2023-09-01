package gcs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"

	"cloud.google.com/go/storage"
)

type Client struct {
	client       *storage.Client
	bucketName   string
	bucketHandle *storage.BucketHandle
}

func NewClient(client *storage.Client, bucket string) (*Client, error) {
	if bucket == "" {
		return nil, errors.New("provided bucket variable is empty")
	}
	return &Client{
		client:       client,
		bucketName:   bucket,
		bucketHandle: client.Bucket(bucket),
	}, nil
}

func (c *Client) Get(ctx context.Context, filename string) ([]byte, error) {
	reader, err := c.bucketHandle.Object(filename).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Client) Store(ctx context.Context, data io.Reader, filename string) error {
	writer := c.bucketHandle.Object(filename).NewWriter(ctx)
	defer writer.Close()
	amount, err := io.Copy(writer, data)
	slog.Info("Copying data", "filename", filename, "bucket", c.bucketName, "amount", amount)
	return err
}

func (c *Client) GetLocation() string {
	return c.bucketName
}
