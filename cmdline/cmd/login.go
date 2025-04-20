/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/apcichewicz/uploader/config"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var oauth2Config *oauth2.Config

func init() {
	InitConfig()
}

func InitConfig() {
	config.InitConfig()
	oauth2Config = &oauth2.Config{
		ClientID:    "UtV37wxFfxPhNDYnvhEqhEZOQBMEJJCFAsvxfLeo",
		RedirectURL: "http://127.0.0.1:8080/redirect",
		Scopes:      []string{"user.read", "user.write", "openid"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "http://localhost:9000/application/o/authorize/",
			TokenURL:  "http://localhost:9000/application/o/token/",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the application",
	Long:  `Login to the application using oauth2`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		state := uuid.New().String()
		authURL := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

		fmt.Println("Opening browser for authentication...")
		code := make(chan string)
		defer close(code)
		errCh := make(chan error)
		defer close(errCh)

		go startCallbackServer(ctx, code, errCh)
		fmt.Println("Please visit the following URL to authenticate:")
		fmt.Println(authURL)

		select {
		case <-ctx.Done():
			log.Fatal("Timeout waiting for code")
		case receivedCode := <-code:
			token, err := oauth2Config.Exchange(context.Background(), receivedCode)
			if err != nil {
				log.Fatal("Failed to exchange code for token:", err)
			}
			config.AppConfig.SetToken(token)
			fmt.Println("Successfully logged in!")
			return
		}
	},
}

func startCallbackServer(ctx context.Context, code chan string, errCh chan error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", handleCallback(code))
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	go func() {
		select {
		case err := <-errCh:
			log.Fatal("Failed to start callback server:", err)
		case <-ctx.Done():
			server.Shutdown(context.Background())
		}
	}()

}

func handleCallback(ch chan<- string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		ch <- code
		fmt.Println(code)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<h1>Successfully logged in!</h1>\n<p>You can close this window now.</p>"))

	}
}

func init() {
	rootCmd.AddCommand(loginCmd)

}
