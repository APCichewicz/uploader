package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/apcichewicz/uploade-service/upload_service"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
)

type Server struct {
	upload_service *upload_service.Uploader
	jwks           *keyfunc.JWKS
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

func (s *Server) WsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Upgrading to WebSocket")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade to WebSocket:", err)
		return
	}
	defer conn.Close()

	blobName := r.URL.Query().Get("blob")
	pipeWriter, resultch := s.upload_service.NewAsyncUploader(r.Context(), blobName)

	// Set a handler for close frames
	conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("Received close frame with code %d: %s", code, text)
		pipeWriter.Close()
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if len(message) == 0 {
			log.Println("Received empty message, closing pipe")
			pipeWriter.Close()
			break
		}
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Println("Normal closure from client")
			} else {
				log.Printf("Read error: %v", err)
			}
			pipeWriter.Close() // Close pipe on any error
			break
		}
		log.Printf("Read %d bytes", len(message))
		if _, err := pipeWriter.Write(message); err != nil {
			log.Printf("Error writing to pipe: %v", err)
			break
		}
		log.Printf("Wrote %d bytes to pipe", len(message))
	}

	// Wait for upload to complete
	for result := range resultch {
		if result.Error != nil {
			log.Println("Failed to upload:", result.Error)
			conn.WriteMessage(websocket.TextMessage, []byte("Upload failed: "+result.Error.Error()))
		} else {
			log.Println("Uploaded:", result.BlobName)
			conn.WriteMessage(websocket.TextMessage, []byte("Upload successful: "+result.BlobName))
		}
	}

	// Send a close message to the client
	conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Upload complete"),
		time.Now().Add(time.Second),
	)
}
func setupJWKS(jwksURL string) (*keyfunc.JWKS, error) {
	// Create context for JWKS operations
	ctx := context.Background()

	// Configure options for the JWKS provider
	options := keyfunc.Options{
		Ctx: ctx,
		RefreshErrorHandler: func(err error) {
			log.Printf("Error refreshing JWKS: %v", err)
		},
		RefreshInterval:   time.Hour,        // Check for new keys every hour
		RefreshRateLimit:  time.Minute * 5,  // Limit refresh rate
		RefreshTimeout:    time.Second * 10, // Timeout for refresh requests
		RefreshUnknownKID: true,             // Try to refresh if we encounter an unknown key ID
		JWKUseWhitelist:   []keyfunc.JWKUse{keyfunc.JWKUse("sig"), keyfunc.JWKUse("enc")},
	}

	// Get the JWKS from Authentik
	return keyfunc.Get(jwksURL, options)
}

func validateToken(tokenString string, jwks *keyfunc.JWKS) (*jwt.Token, jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, jwks.Keyfunc)

	if err != nil {
		return nil, nil, err
	}

	// Verify it's a valid token
	if !token.Valid {
		return nil, nil, fmt.Errorf("invalid token")
	}

	// Get the claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, nil, fmt.Errorf("invalid claims")
	}

	return token, claims, nil
}

func NewServer(upload_service *upload_service.Uploader, jwks_url string) *Server {
	jwks, err := setupJWKS(jwks_url)
	if err != nil {
		log.Fatal("Failed to setup JWKS: ", err)
	}
	return &Server{upload_service: upload_service, jwks: jwks}
}

func (s *Server) Start() {
	log.Println("Starting server on port 8090")
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Upload request received")
		s.AuthMiddleware(http.HandlerFunc(s.WsHandler)).ServeHTTP(w, r)
	})
	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			fmt.Println("No token found")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		_, _, err := validateToken(tokenString, s.jwks)
		if err != nil {
			fmt.Println("Token validation failed")
			fmt.Println(err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		fmt.Println("Token validated")
		next.ServeHTTP(w, r)
	})
}
