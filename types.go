package main

import (
	"net/http"
	"os"
	"time"

	attackdetectionpb "request-extractor/proto"

	"google.golang.org/grpc"
)

const (
	defaultMLRPCAddr     = "localhost:9000"
	defaultListenAddr    = ":8081"
	defaultLogPath       = "requests.log"
	defaultMaxBodyBytes  = 1 << 20
	defaultMLRPCTimeout  = 2 * time.Second
	defaultTelegramDelay = 8 * time.Second
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

type Config struct {
	ListenAddr        string
	LogPath           string
	MLRPCAddr         string
	MLRPCTimeout      time.Duration
	MaxBodyBytes      int
	TelegramBotToken  string
	TelegramChatID    string
	TelegramHTTPDelay time.Duration
}

type App struct {
	cfg        Config
	logFile    *os.File
	mlConn     *grpc.ClientConn
	mlClient   attackdetectionpb.AttackDetectionClient
	httpClient *http.Client
}
