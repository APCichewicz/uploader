package upload_service

import (
	"context"
	"fmt"
	"io"
	"os"

	// azblob
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type UploadResult struct {
	BlobName string
	Response *azblob.UploadStreamResponse
	Error    error
}

type Uploader struct {
	client *azblob.Client
}

func NewUploader(ctx context.Context) (*Uploader, error) {
	connString := os.Getenv("AZURE_BLOB_CONNECTION_STRING")
	if connString == "" {
		return nil, fmt.Errorf("AZURE_BLOB_CONNECTION_STRING is not set")
	}
	blobclient, err := azblob.NewClientFromConnectionString(connString, &azblob.ClientOptions{})
	if err != nil {
		return nil, err
	}
	return &Uploader{client: blobclient}, nil
}

func (u *Uploader) NewAsyncUploader(ctx context.Context, blobName string) (*io.PipeWriter, <-chan UploadResult) {
	containerName := os.Getenv("AZURE_BLOB_CONTAINER_NAME")
	reader, writer := io.Pipe()
	resultch := make(chan UploadResult)

	go func() {
		defer close(resultch)
		defer reader.Close()
		response, err := u.client.UploadStream(ctx, containerName, blobName, reader, nil)
		resultch <- UploadResult{BlobName: blobName, Response: &response, Error: err}
	}()
	return writer, resultch
}
