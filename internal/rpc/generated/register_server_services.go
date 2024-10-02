// Code generated by '{{GENERATOR}}'. DO NOT EDIT.

package generated

import (
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"google.golang.org/grpc"
)

func RegisterServerServices(grpcServer *grpc.Server, reqProcessor rpc.ExternalRequestProcessor) {
	registerAccommodationProductInfoServiceV1Server(grpcServer, reqProcessor)
	registerAccommodationProductListServiceV1Server(grpcServer, reqProcessor)
	registerAccommodationSearchServiceV1Server(grpcServer, reqProcessor)
	registerAccommodationProductInfoServiceV2Server(grpcServer, reqProcessor)
	registerAccommodationProductListServiceV2Server(grpcServer, reqProcessor)
	registerAccommodationSearchServiceV2Server(grpcServer, reqProcessor)
	registerActivityProductInfoServiceV1Server(grpcServer, reqProcessor)
	registerActivityProductListServiceV1Server(grpcServer, reqProcessor)
	registerActivitySearchServiceV1Server(grpcServer, reqProcessor)
	registerActivityProductInfoServiceV2Server(grpcServer, reqProcessor)
	registerActivityProductListServiceV2Server(grpcServer, reqProcessor)
	registerActivitySearchServiceV2Server(grpcServer, reqProcessor)
	registerMintServiceV1Server(grpcServer, reqProcessor)
	registerValidationServiceV1Server(grpcServer, reqProcessor)
	registerMintServiceV2Server(grpcServer, reqProcessor)
	registerValidationServiceV2Server(grpcServer, reqProcessor)
	registerCountryEntryRequirementsServiceV1Server(grpcServer, reqProcessor)
	registerCountryEntryRequirementsServiceV2Server(grpcServer, reqProcessor)
	registerInsuranceProductInfoServiceV1Server(grpcServer, reqProcessor)
	registerInsuranceProductListServiceV1Server(grpcServer, reqProcessor)
	registerInsuranceSearchServiceV1Server(grpcServer, reqProcessor)
	registerGetNetworkFeeServiceV1Server(grpcServer, reqProcessor)
	registerGetPartnerConfigurationServiceV1Server(grpcServer, reqProcessor)
	registerGetPartnerConfigurationServiceV2Server(grpcServer, reqProcessor)
	registerPingServiceV1Server(grpcServer, reqProcessor)
	registerSeatMapAvailabilityServiceV1Server(grpcServer, reqProcessor)
	registerSeatMapServiceV1Server(grpcServer, reqProcessor)
	registerSeatMapAvailabilityServiceV2Server(grpcServer, reqProcessor)
	registerSeatMapServiceV2Server(grpcServer, reqProcessor)
	registerTransportSearchServiceV1Server(grpcServer, reqProcessor)
	registerTransportSearchServiceV2Server(grpcServer, reqProcessor)
}
