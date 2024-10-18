// Code generated by './scripts/generate_grpc_service_handlers.sh'. DO NOT EDIT.
// template: templates/server.go.tpl

package generated

import (
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v1/insurancev1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"

	"google.golang.org/grpc"
)

var _ insurancev1grpc.InsuranceProductInfoServiceServer = (*insurancev1InsuranceProductInfoServiceServer)(nil)

type insurancev1InsuranceProductInfoServiceServer struct {
	reqProcessor rpc.ExternalRequestProcessor
}

func registerInsuranceProductInfoServiceV1Server(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	insurancev1grpc.RegisterInsuranceProductInfoServiceServer(grpcServer, &insurancev1InsuranceProductInfoServiceServer{reqProcessor})
}
