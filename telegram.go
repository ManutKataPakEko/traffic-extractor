package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"sort"
	"strings"
)

func isAttackPrediction(prediction string) bool {
	return prediction == "Attack"
}

func (a *App) sendAttackAlert(payload EnrichedPayload) {
	if a.cfg.TelegramBotToken == "" || a.cfg.TelegramChatID == "" {
		return
	}

	headersAnalysis := formatHeadersForAlert(payload.Headers, 25)
	bodyAnalysis := truncateForAlert(payload.Body, 1200)

	lines := []string{
		"<b>ALERTA ALERTA</b>",
		"Potential intrusion attempt detected by OJS payload classifier.",
		"",
		"<b>Prediction:</b> <code>" + html.EscapeString(payload.Prediction) + "</code>",
		"<b>Method:</b> <code>" + html.EscapeString(payload.Method) + "</code>",
		"<b>Path:</b> <code>" + html.EscapeString(payload.Path) + "</code>",
		"<b>IP:</b> <code>" + html.EscapeString(payload.IP) + "</code>",
		"<b>Timestamp:</b> " + html.EscapeString(payload.Timestamp),
		"",
		"<b>Headers analysis</b>",
		"<pre>" + html.EscapeString(headersAnalysis) + "</pre>",
		"",
		"<b>Body analysis</b>",
		"<pre>" + html.EscapeString(bodyAnalysis) + "</pre>",
	}

	message := strings.Join(lines, "\n")
	if err := a.sendTelegramMessage(message); err != nil {
		log.Printf("failed to send Telegram alert: %v", err)
	}
}

func formatHeadersForAlert(headers map[string]string, maxHeaders int) string {
	if len(headers) == 0 {
		return "(no headers)"
	}
	if maxHeaders <= 0 {
		maxHeaders = 1
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, min(len(keys), maxHeaders)+1)
	for i, key := range keys {
		if i >= maxHeaders {
			lines = append(lines, "... (truncated)")
			break
		}
		lines = append(lines, key+": "+headers[key])
	}

	return strings.Join(lines, "\n")
}

func truncateForAlert(content string, maxLen int) string {
	if content == "" {
		return "(empty body)"
	}
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n... (truncated)"
}

func (a *App) sendTelegramMessage(text string) error {
	url := "https://api.telegram.org/bot" + a.cfg.TelegramBotToken + "/sendMessage"

	body := map[string]any{
		"chat_id":    a.cfg.TelegramChatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API status: %s", resp.Status)
	}

	var telegramResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return err
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return nil
}
