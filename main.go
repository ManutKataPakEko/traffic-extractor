package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	attackdetectionpb "request-extractor/proto"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RequestPayload struct {
	Timestamp string            `json:"timestamp"`
	IP        string            `json:"ip"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Query     string            `json:"query"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
}

type EnrichedPayload struct {
	RequestPayload
	Prediction string `json:"prediction,omitempty"`
}

var logFile *os.File
var sensitiveRegex = regexp.MustCompile(`(?i)(password|password2|pass|passwd)=([^&]+)`)
var mlConn *grpc.ClientConn
var mlClient attackdetectionpb.AttackDetectionClient

func main() {
	// Initialize log file (for file logging mode)
	var err error
	logFile, err = os.OpenFile(
		"requests.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("failed to load .env: %v", err)
	}

	if err := initMLClient(); err != nil {
		log.Fatalf("failed to initialize ML gRPC client: %v", err)
	}
	defer mlConn.Close()

	r := fiber.New()

	r.All("/*", func(c *fiber.Ctx) error {

		// Limit body size to 1MB
		bodyBytes := c.Body()
		if len(bodyBytes) > (1 << 20) {
			return c.SendStatus(http.StatusRequestEntityTooLarge)
		}

		// Extract headers
		headers := make(map[string]string)
		c.Request().Header.VisitAll(func(key, value []byte) {
			headers[string(key)] = string(value)
		})

		// Remove sensitive headers
		delete(headers, "Authorization")
		delete(headers, "Cookie")

		path := c.Path()

		if original := c.Get("X-Original-Uri"); original != "" {
			path = original
		}

		payload := RequestPayload{
			Timestamp: time.Now().Format(time.RFC3339),
			IP:        c.IP(),
			Method:    c.Method(),
			Path:      path,
			Query:     string(c.Request().URI().QueryString()),
			Headers:   headers,
			Body:      sanitizeBody(string(bodyBytes)),
		}

		enrichedPayload := forwardToML(payload)
		writeToFile(enrichedPayload)

		// For now default to debug
		debugPrint(enrichedPayload)

		return c.SendStatus(http.StatusOK)
	})

	r.Listen(":8081")
}

func initMLClient() error {
	mlAddr := os.Getenv("ML_RPC_ADDR")
	if mlAddr == "" {
		mlAddr = "localhost:9000"
	}

	conn, err := grpc.NewClient(mlAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	mlConn = conn
	mlClient = attackdetectionpb.NewAttackDetectionClient(conn)
	return nil
}

// Send to ML Engine (gRPC Predict)
func forwardToML(payload RequestPayload) EnrichedPayload {
	enriched := EnrichedPayload{RequestPayload: payload}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := mlClient.Predict(ctx, &attackdetectionpb.PredictRequest{
		Timestamp: payload.Timestamp,
		Ip:        payload.IP,
		Method:    payload.Method,
		Path:      payload.Path,
		Query:     payload.Query,
		Headers:   payload.Headers,
		Body:      payload.Body,
	})
	if err != nil {
		log.Println("ML RPC error:", err)
		return enriched
	}

	if resp != nil {
		enriched.Prediction = resp.GetPrediction()
	}

	return enriched
}

// Write JSON Line to File
func writeToFile(payload EnrichedPayload) {

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	_, err = logFile.Write(jsonData)
	if err != nil {
		log.Println("File write error:", err)
		return
	}

	logFile.Write([]byte("\n"))
}

// Debug Mode (Print to Terminal)
func debugPrint(payload EnrichedPayload) {

	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	fmt.Println(string(jsonData))
	fmt.Println("---------------------------------------------------")
}

func sanitizeBody(body string) string {
	return sensitiveRegex.ReplaceAllString(body, "$1=***REDACTED***")
}
