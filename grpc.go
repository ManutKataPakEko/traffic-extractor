package main

import (
	"context"
	"fmt"
	"log"

	attackdetectionpb "request-extractor/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newMLClient(addr string) (*grpc.ClientConn, attackdetectionpb.AttackDetectionClient, error) {
	mlConn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("init ML gRPC client: %w", err)
	}

	return mlConn, attackdetectionpb.NewAttackDetectionClient(mlConn), nil
}

func (a *App) forwardToML(payload RequestPayload) EnrichedPayload {
	enriched := EnrichedPayload{RequestPayload: payload}

	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.MLRPCTimeout)
	defer cancel()

	resp, err := a.mlClient.Predict(ctx, &attackdetectionpb.PredictRequest{
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
