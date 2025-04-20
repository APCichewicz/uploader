/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	// import config
	"github.com/apcichewicz/uploader/config"
	// import websocket
	"github.com/gorilla/websocket"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file to Azure Blob Storage",
	Long:  `Upload a file to Azure Blob Storage.`,
	Run: func(cmd *cobra.Command, args []string) {
		filename, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatalf("Failed to get file: %v", err)
		}
		blob, err := cmd.Flags().GetString("blob")
		if err != nil {
			log.Fatalf("Failed to get blob: %v", err)
		}
		if filename == "" {
			log.Fatalf("File is required")
		}
		if blob == "" {
			log.Fatalf("Blob is required")
		}

		token := config.AppConfig.GetToken()
		if token == "" {
			log.Fatalf("User must login first")
		}

		// connect to the upload service
		header := http.Header{}
		header.Add("Authorization", "Bearer "+token)
		dialer := websocket.Dialer{}
		conn, _, err := dialer.Dial(fmt.Sprintf("ws://localhost:8090/upload?blob=%s", blob), header)
		if err != nil {
			log.Fatalf("Failed to connect to upload service: %v", err)
		}
		defer conn.Close()
		// send the file to the upload service
		fmt.Println("Sending file to upload service")
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		buffer := make([]byte, 1024)
		for {
			fmt.Println("Reading file")
			n, err := reader.Read(buffer)
			if err != nil {
				if err == io.EOF {
					fmt.Println("EOF")
					err := conn.WriteMessage(websocket.BinaryMessage, []byte{})
					if err != nil {
						log.Fatalf("Failed to write message: %v", err)
					}
					break
				}
				log.Fatalf("Failed to read file: %v", err)
			}
			if n > 0 {
				fmt.Println("Writing message")
				fmt.Println(buffer[:n])
				err := conn.WriteMessage(websocket.BinaryMessage, buffer[:n])
				if err != nil {
					log.Fatalf("Failed to write message: %v", err)
				}
			}
		}
		// wait for the server to close the connection
		_, _, err = conn.ReadMessage()
		if err != nil {
			log.Fatalf("Failed to read message: %v", err)
		}
		fmt.Println("Connection closed")
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	config.InitConfig()
	uploadCmd.Flags().StringP("file", "f", "", "The file to upload")
	uploadCmd.Flags().StringP("blob", "b", "", "The blob name")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// uploadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// uploadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
