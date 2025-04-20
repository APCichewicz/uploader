package main

import (
	"context"
	"log"

	"github.com/apcichewicz/uploade-service/server"
	"github.com/apcichewicz/uploade-service/upload_service"
)

func main() {
	upload_service, err := upload_service.NewUploader(context.Background())
	if err != nil {
		log.Fatalf("Failed to create upload service: %v", err)
	}
	server := server.NewServer(upload_service, "http://localhost:9000/application/o/uploader/jwks/")
	server.Start()
}
