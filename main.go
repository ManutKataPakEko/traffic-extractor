package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
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

var logFile *os.File
var sensitiveRegex = regexp.MustCompile(`(?i)(password|password2|pass|passwd)=([^&]+)`)

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

		// forwardToML(payload)
		writeToFile(payload)

		// For now default to debug
		debugPrint(payload)

		return c.SendStatus(http.StatusOK)
	})

	r.Listen(":8081")
}

// Send to ML Engine (HTTP POST)
func forwardToML(payload RequestPayload) {

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	resp, err := http.Post(
		"http://localhost:9000/analyze",
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		log.Println("ML POST error:", err)
		return
	}
	defer resp.Body.Close()
}

// Write JSON Line to File
func writeToFile(payload RequestPayload) {

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
func debugPrint(payload RequestPayload) {

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
