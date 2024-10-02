// Code generated by '{{GENERATOR}}'. DO NOT EDIT.

package generated

import (
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"google.golang.org/grpc"
)

func RegisterServices(rpcConn *grpc.ClientConn, serviceNames map[string]struct{}) map[types.MessageType]rpc.Service {
	services := make(map[types.MessageType]rpc.Service, len(serviceNames))

	if _, ok := serviceNames[PingServiceV1]; ok {
		services[PingServiceV1Request] = rpc.NewService(NewPingServiceV1(rpcConn), PingServiceV1)
		delete(serviceNames, PingServiceV1)
	}
	if srvName, ok := serviceNames[CountryEntryRequirementsServiceV2Request]; ok {
		services[CountryEntryRequirementsServiceV2Request] = rpc.NewService(NewCountryEntryRequirementsServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[CountryEntryRequirementsServiceV1Request]; ok {
		services[CountryEntryRequirementsServiceV1Request] = rpc.NewService(NewCountryEntryRequirementsServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[AccommodationProductListServiceV2Request]; ok {
		services[AccommodationProductListServiceV2Request] = rpc.NewService(NewAccommodationProductListServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[AccommodationSearchServiceV2Request]; ok {
		services[AccommodationSearchServiceV2Request] = rpc.NewService(NewAccommodationSearchServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[AccommodationProductInfoServiceV2Request]; ok {
		services[AccommodationProductInfoServiceV2Request] = rpc.NewService(NewAccommodationProductInfoServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[AccommodationProductListServiceV1Request]; ok {
		services[AccommodationProductListServiceV1Request] = rpc.NewService(NewAccommodationProductListServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[AccommodationSearchServiceV1Request]; ok {
		services[AccommodationSearchServiceV1Request] = rpc.NewService(NewAccommodationSearchServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[AccommodationProductInfoServiceV1Request]; ok {
		services[AccommodationProductInfoServiceV1Request] = rpc.NewService(NewAccommodationProductInfoServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[ActivityProductListServiceV2Request]; ok {
		services[ActivityProductListServiceV2Request] = rpc.NewService(NewActivityProductListServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[ActivitySearchServiceV2Request]; ok {
		services[ActivitySearchServiceV2Request] = rpc.NewService(NewActivitySearchServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[ActivityProductInfoServiceV2Request]; ok {
		services[ActivityProductInfoServiceV2Request] = rpc.NewService(NewActivityProductInfoServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[ActivityProductListServiceV1Request]; ok {
		services[ActivityProductListServiceV1Request] = rpc.NewService(NewActivityProductListServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[ActivitySearchServiceV1Request]; ok {
		services[ActivitySearchServiceV1Request] = rpc.NewService(NewActivitySearchServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[ActivityProductInfoServiceV1Request]; ok {
		services[ActivityProductInfoServiceV1Request] = rpc.NewService(NewActivityProductInfoServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[InsuranceProductListServiceV1Request]; ok {
		services[InsuranceProductListServiceV1Request] = rpc.NewService(NewInsuranceProductListServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[InsuranceSearchServiceV1Request]; ok {
		services[InsuranceSearchServiceV1Request] = rpc.NewService(NewInsuranceSearchServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[InsuranceProductInfoServiceV1Request]; ok {
		services[InsuranceProductInfoServiceV1Request] = rpc.NewService(NewInsuranceProductInfoServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[TransportSearchServiceV2Request]; ok {
		services[TransportSearchServiceV2Request] = rpc.NewService(NewTransportSearchServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[TransportSearchServiceV1Request]; ok {
		services[TransportSearchServiceV1Request] = rpc.NewService(NewTransportSearchServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[GetNetworkFeeServiceV1Request]; ok {
		services[GetNetworkFeeServiceV1Request] = rpc.NewService(NewGetNetworkFeeServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[SeatMapServiceV2Request]; ok {
		services[SeatMapServiceV2Request] = rpc.NewService(NewSeatMapServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[SeatMapAvailabilityServiceV2Request]; ok {
		services[SeatMapAvailabilityServiceV2Request] = rpc.NewService(NewSeatMapAvailabilityServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[SeatMapServiceV1Request]; ok {
		services[SeatMapServiceV1Request] = rpc.NewService(NewSeatMapServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[SeatMapAvailabilityServiceV1Request]; ok {
		services[SeatMapAvailabilityServiceV1Request] = rpc.NewService(NewSeatMapAvailabilityServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[MintServiceV2Request]; ok {
		services[MintServiceV2Request] = rpc.NewService(NewMintServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[ValidationServiceV2Request]; ok {
		services[ValidationServiceV2Request] = rpc.NewService(NewValidationServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[MintServiceV1Request]; ok {
		services[MintServiceV1Request] = rpc.NewService(NewMintServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[ValidationServiceV1Request]; ok {
		services[ValidationServiceV1Request] = rpc.NewService(NewValidationServiceV1(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[GetPartnerConfigurationServiceV2Request]; ok {
		services[GetPartnerConfigurationServiceV2Request] = rpc.NewService(NewGetPartnerConfigurationServiceV2(rpcConn), srvName)
	}
	if srvName, ok := serviceNames[GetPartnerConfigurationServiceV1Request]; ok {
		services[GetPartnerConfigurationServiceV1Request] = rpc.NewService(NewGetPartnerConfigurationServiceV1(rpcConn), srvName)
	}
	return services
}
