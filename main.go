package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
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

	r := gin.Default()

	r.Any("/*path", func(c *gin.Context) {

		// Limit body size to 1MB
		c.Request.Body = http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			1<<20,
		)

		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Extract headers
		headers := make(map[string]string)
		for k, v := range c.Request.Header {
			headers[k] = v[0]
		}

		// Remove sensitive headers
		delete(headers, "Authorization")
		delete(headers, "Cookie")

		payload := RequestPayload{
			Timestamp: time.Now().Format(time.RFC3339),
			IP:        c.ClientIP(),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Query:     c.Request.URL.RawQuery,
			Headers:   headers,
			Body:      string(bodyBytes),
		}

		// forwardToML(payload)
		writeToFile(payload)

		// For now default to debug
		debugPrint(payload)

		c.Status(http.StatusOK)
	})

	r.Run(":8081")
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
