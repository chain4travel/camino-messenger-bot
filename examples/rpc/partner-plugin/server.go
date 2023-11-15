package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"github.com/chain4travel/camino-messenger-bot/proto/pb/messages"
	"google.golang.org/grpc"
)

type partnerPlugin struct {
	messages.FlightSearchServiceServer
}

func (p *partnerPlugin) Search(ctx context.Context, request *messages.FlightSearchRequest) (*messages.FlightSearchResponse, error) {

	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)
	response := messages.FlightSearchResponse{
		Header:            nil,
		Context:           fmt.Sprintf("Response to: %s", md.RequestID),
		Errors:            "",
		Warnings:          "",
		SupplierCode:      "",
		ExternalSessionId: "",
		SearchId:          md.RequestID,
		Options:           nil,
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func main() {
	grpcServer := grpc.NewServer()
	messages.RegisterFlightSearchServiceServer(grpcServer, &partnerPlugin{})

	port := 55555
	var err error
	p, found := os.LookupEnv("PORT")
	if found {
		port, err = strconv.Atoi(p)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Starting server on port: %d", port)
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer.Serve(lis)
}
