package main

import (
	"os"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var sensitiveRegex = regexp.MustCompile(`(?i)(password|password2|pass|passwd)=([^&]+)`)

func sanitizeBody(body string) string {
	return sensitiveRegex.ReplaceAllString(body, "$1=***REDACTED***")
}

func extractFilteredHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	delete(headers, "Authorization")
	delete(headers, "Cookie")
	return headers
}

func resolvePath(c *fiber.Ctx) string {
	if original := c.Get("X-Original-Uri"); original != "" {
		return original
	}
	return c.Path()
}

func resolveClientIP(c *fiber.Ctx) string {
	if cfIP := c.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}

	if xff := c.Get("X-Forwarded-For"); xff != "" {
		// XFF can contain multiple IPs: client, proxy1, proxy2
		// take the first one
		return strings.Split(xff, ",")[0]
	}

	if xrip := c.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	return c.IP()
}

func getenvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
