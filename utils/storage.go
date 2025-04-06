package utils

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
)

func UploadToBucket(file multipart.File, postName string, ctx context.Context, uploadTimeout time.Duration) (string, error) {
	bucketName := "circle_app_posts"

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	o := client.Bucket(bucketName).Object(postName)

	wc := o.NewWriter(ctx)
	if _, err = io.Copy(wc, file); err != nil {
		wc.Close()
		return "", err
	}

	if err := wc.Close(); err != nil {
		return "", err
	}

	fmt.Printf("Blob uploaded successfully: %s", postName)
	return postName, nil
}

func GenerateGetSignedURL(object string, context context.Context) (string, error) {
	bucket := os.Getenv("BUCKET_NAME")
	sakeyFile := "./sa-cred.json"

	saKey, err := os.ReadFile(sakeyFile)
	if err != nil {
		return "", fmt.Errorf("failed to read service account key")
	}

	cfg, err := google.JWTConfigFromJSON(saKey)
	if err != nil {
		return "", fmt.Errorf("failed to read config file with service account key")
	}

	client, err := storage.NewClient(context)
	if err != nil {
		return "", err
	}
	defer client.Close()

	opts := &storage.SignedURLOptions{
		GoogleAccessID: cfg.Email,
		PrivateKey:     cfg.PrivateKey,
		Scheme:         storage.SigningSchemeV4,
		Method:         "GET",
		Expires:        time.Now().Add(15 * time.Minute),
	}

	url, err := client.Bucket(bucket).SignedURL(object, opts)
	if err != nil {
		return "", fmt.Errorf("Bucket(%q).SignedURL: %w", bucket, err)
	}

	return url, nil
}
