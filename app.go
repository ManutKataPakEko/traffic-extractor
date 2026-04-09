package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

func (a *App) close() {
	if a.mlConn != nil {
		_ = a.mlConn.Close()
	}
	if a.logFile != nil {
		_ = a.logFile.Close()
	}
}

func (a *App) registerRoutes(router *fiber.App) {
	router.All("/*", a.handleRequest)
}

func (a *App) handleRequest(c *fiber.Ctx) error {
	payload, status, err := a.buildPayload(c)
	if err != nil {
		if status != 0 {
			return c.SendStatus(status)
		}
		log.Printf("build payload error: %v", err)
		return c.SendStatus(http.StatusBadRequest)
	}

	enriched := a.forwardToML(payload)
	a.writeToFile(enriched)

	if isAttackPrediction(enriched.Prediction) {
		go a.sendAttackAlert(enriched)
	}

	debugPrint(enriched)
	return c.SendStatus(http.StatusOK)
}

func (a *App) buildPayload(c *fiber.Ctx) (RequestPayload, int, error) {
	bodyBytes := c.Body()
	if len(bodyBytes) > a.cfg.MaxBodyBytes {
		return RequestPayload{}, http.StatusRequestEntityTooLarge, errors.New("body too large")
	}

	return RequestPayload{
		Timestamp: time.Now().Format(time.RFC3339),
		IP:        resolveClientIP(c),
		Method:    c.Method(),
		Path:      resolvePath(c),
		Query:     string(c.Request().URI().QueryString()),
		Headers:   extractFilteredHeaders(c),
		Body:      sanitizeBody(string(bodyBytes)),
	}, 0, nil
}

func (a *App) writeToFile(payload EnrichedPayload) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	if _, err := a.logFile.Write(jsonData); err != nil {
		log.Println("File write error:", err)
		return
	}

	if _, err := a.logFile.Write([]byte("\n")); err != nil {
		log.Println("File write error:", err)
	}
}

func debugPrint(payload EnrichedPayload) {
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Println("JSON marshal error:", err)
		return
	}

	fmt.Println(string(jsonData))
	fmt.Println("---------------------------------------------------")
}
