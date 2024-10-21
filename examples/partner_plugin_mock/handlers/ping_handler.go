package handlers_mock

import (
	"context"
	"fmt"
	"log"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"

	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
)

type PingResponseHandler interface {
	Ping(ctx context.Context, request *pingv1.PingRequest) (*pingv1.PingResponse, error)
}

func NewPingResponseHandler() PingResponseHandler {
	return &pingResponseHandler{}
}

type pingResponseHandler struct{}

func (p *pingResponseHandler) Ping(ctx context.Context, request *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (Ping)", md.RequestID)

	return &pingv1.PingResponse{
		Header:      nil,
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", request.PingMessage, md.RequestID),
	}, nil
}
