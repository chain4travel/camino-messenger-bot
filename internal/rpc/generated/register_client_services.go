// Code generated by '{{GENERATOR}}'. DO NOT EDIT.

package generated

import (
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	"github.com/chain4travel/camino-messenger-bot/internal/rpc"
	"google.golang.org/grpc"
)

func RegisterClientServices(rpcConn *grpc.ClientConn, serviceNames map[string]struct{}) map[types.MessageType]rpc.Service {
	services := make(map[types.MessageType]rpc.Service, len(serviceNames))

	if _, ok := serviceNames[AccommodationProductInfoServiceV1]; ok {
		services[AccommodationProductInfoServiceV1Request] = rpc.NewService(NewAccommodationProductInfoServiceV1(rpcConn), AccommodationProductInfoServiceV1)
		delete(serviceNames, AccommodationProductInfoServiceV1)
	}
	if _, ok := serviceNames[AccommodationProductListServiceV1]; ok {
		services[AccommodationProductListServiceV1Request] = rpc.NewService(NewAccommodationProductListServiceV1(rpcConn), AccommodationProductListServiceV1)
		delete(serviceNames, AccommodationProductListServiceV1)
	}
	if _, ok := serviceNames[AccommodationSearchServiceV1]; ok {
		services[AccommodationSearchServiceV1Request] = rpc.NewService(NewAccommodationSearchServiceV1(rpcConn), AccommodationSearchServiceV1)
		delete(serviceNames, AccommodationSearchServiceV1)
	}
	if _, ok := serviceNames[AccommodationProductInfoServiceV2]; ok {
		services[AccommodationProductInfoServiceV2Request] = rpc.NewService(NewAccommodationProductInfoServiceV2(rpcConn), AccommodationProductInfoServiceV2)
		delete(serviceNames, AccommodationProductInfoServiceV2)
	}
	if _, ok := serviceNames[AccommodationProductListServiceV2]; ok {
		services[AccommodationProductListServiceV2Request] = rpc.NewService(NewAccommodationProductListServiceV2(rpcConn), AccommodationProductListServiceV2)
		delete(serviceNames, AccommodationProductListServiceV2)
	}
	if _, ok := serviceNames[AccommodationSearchServiceV2]; ok {
		services[AccommodationSearchServiceV2Request] = rpc.NewService(NewAccommodationSearchServiceV2(rpcConn), AccommodationSearchServiceV2)
		delete(serviceNames, AccommodationSearchServiceV2)
	}
	if _, ok := serviceNames[ActivityProductInfoServiceV1]; ok {
		services[ActivityProductInfoServiceV1Request] = rpc.NewService(NewActivityProductInfoServiceV1(rpcConn), ActivityProductInfoServiceV1)
		delete(serviceNames, ActivityProductInfoServiceV1)
	}
	if _, ok := serviceNames[ActivityProductListServiceV1]; ok {
		services[ActivityProductListServiceV1Request] = rpc.NewService(NewActivityProductListServiceV1(rpcConn), ActivityProductListServiceV1)
		delete(serviceNames, ActivityProductListServiceV1)
	}
	if _, ok := serviceNames[ActivitySearchServiceV1]; ok {
		services[ActivitySearchServiceV1Request] = rpc.NewService(NewActivitySearchServiceV1(rpcConn), ActivitySearchServiceV1)
		delete(serviceNames, ActivitySearchServiceV1)
	}
	if _, ok := serviceNames[ActivityProductInfoServiceV2]; ok {
		services[ActivityProductInfoServiceV2Request] = rpc.NewService(NewActivityProductInfoServiceV2(rpcConn), ActivityProductInfoServiceV2)
		delete(serviceNames, ActivityProductInfoServiceV2)
	}
	if _, ok := serviceNames[ActivityProductListServiceV2]; ok {
		services[ActivityProductListServiceV2Request] = rpc.NewService(NewActivityProductListServiceV2(rpcConn), ActivityProductListServiceV2)
		delete(serviceNames, ActivityProductListServiceV2)
	}
	if _, ok := serviceNames[ActivitySearchServiceV2]; ok {
		services[ActivitySearchServiceV2Request] = rpc.NewService(NewActivitySearchServiceV2(rpcConn), ActivitySearchServiceV2)
		delete(serviceNames, ActivitySearchServiceV2)
	}
	if _, ok := serviceNames[MintServiceV1]; ok {
		services[MintServiceV1Request] = rpc.NewService(NewMintServiceV1(rpcConn), MintServiceV1)
		delete(serviceNames, MintServiceV1)
	}
	if _, ok := serviceNames[ValidationServiceV1]; ok {
		services[ValidationServiceV1Request] = rpc.NewService(NewValidationServiceV1(rpcConn), ValidationServiceV1)
		delete(serviceNames, ValidationServiceV1)
	}
	if _, ok := serviceNames[MintServiceV2]; ok {
		services[MintServiceV2Request] = rpc.NewService(NewMintServiceV2(rpcConn), MintServiceV2)
		delete(serviceNames, MintServiceV2)
	}
	if _, ok := serviceNames[ValidationServiceV2]; ok {
		services[ValidationServiceV2Request] = rpc.NewService(NewValidationServiceV2(rpcConn), ValidationServiceV2)
		delete(serviceNames, ValidationServiceV2)
	}
	if _, ok := serviceNames[CountryEntryRequirementsServiceV1]; ok {
		services[CountryEntryRequirementsServiceV1Request] = rpc.NewService(NewCountryEntryRequirementsServiceV1(rpcConn), CountryEntryRequirementsServiceV1)
		delete(serviceNames, CountryEntryRequirementsServiceV1)
	}
	if _, ok := serviceNames[CountryEntryRequirementsServiceV2]; ok {
		services[CountryEntryRequirementsServiceV2Request] = rpc.NewService(NewCountryEntryRequirementsServiceV2(rpcConn), CountryEntryRequirementsServiceV2)
		delete(serviceNames, CountryEntryRequirementsServiceV2)
	}
	if _, ok := serviceNames[InsuranceProductInfoServiceV1]; ok {
		services[InsuranceProductInfoServiceV1Request] = rpc.NewService(NewInsuranceProductInfoServiceV1(rpcConn), InsuranceProductInfoServiceV1)
		delete(serviceNames, InsuranceProductInfoServiceV1)
	}
	if _, ok := serviceNames[InsuranceProductListServiceV1]; ok {
		services[InsuranceProductListServiceV1Request] = rpc.NewService(NewInsuranceProductListServiceV1(rpcConn), InsuranceProductListServiceV1)
		delete(serviceNames, InsuranceProductListServiceV1)
	}
	if _, ok := serviceNames[InsuranceSearchServiceV1]; ok {
		services[InsuranceSearchServiceV1Request] = rpc.NewService(NewInsuranceSearchServiceV1(rpcConn), InsuranceSearchServiceV1)
		delete(serviceNames, InsuranceSearchServiceV1)
	}
	if _, ok := serviceNames[GetNetworkFeeServiceV1]; ok {
		services[GetNetworkFeeServiceV1Request] = rpc.NewService(NewGetNetworkFeeServiceV1(rpcConn), GetNetworkFeeServiceV1)
		delete(serviceNames, GetNetworkFeeServiceV1)
	}
	if _, ok := serviceNames[GetPartnerConfigurationServiceV1]; ok {
		services[GetPartnerConfigurationServiceV1Request] = rpc.NewService(NewGetPartnerConfigurationServiceV1(rpcConn), GetPartnerConfigurationServiceV1)
		delete(serviceNames, GetPartnerConfigurationServiceV1)
	}
	if _, ok := serviceNames[GetPartnerConfigurationServiceV2]; ok {
		services[GetPartnerConfigurationServiceV2Request] = rpc.NewService(NewGetPartnerConfigurationServiceV2(rpcConn), GetPartnerConfigurationServiceV2)
		delete(serviceNames, GetPartnerConfigurationServiceV2)
	}
	if _, ok := serviceNames[PingServiceV1]; ok {
		services[PingServiceV1Request] = rpc.NewService(NewPingServiceV1(rpcConn), PingServiceV1)
		delete(serviceNames, PingServiceV1)
	}
	if _, ok := serviceNames[SeatMapAvailabilityServiceV1]; ok {
		services[SeatMapAvailabilityServiceV1Request] = rpc.NewService(NewSeatMapAvailabilityServiceV1(rpcConn), SeatMapAvailabilityServiceV1)
		delete(serviceNames, SeatMapAvailabilityServiceV1)
	}
	if _, ok := serviceNames[SeatMapServiceV1]; ok {
		services[SeatMapServiceV1Request] = rpc.NewService(NewSeatMapServiceV1(rpcConn), SeatMapServiceV1)
		delete(serviceNames, SeatMapServiceV1)
	}
	if _, ok := serviceNames[SeatMapAvailabilityServiceV2]; ok {
		services[SeatMapAvailabilityServiceV2Request] = rpc.NewService(NewSeatMapAvailabilityServiceV2(rpcConn), SeatMapAvailabilityServiceV2)
		delete(serviceNames, SeatMapAvailabilityServiceV2)
	}
	if _, ok := serviceNames[SeatMapServiceV2]; ok {
		services[SeatMapServiceV2Request] = rpc.NewService(NewSeatMapServiceV2(rpcConn), SeatMapServiceV2)
		delete(serviceNames, SeatMapServiceV2)
	}
	if _, ok := serviceNames[TransportSearchServiceV1]; ok {
		services[TransportSearchServiceV1Request] = rpc.NewService(NewTransportSearchServiceV1(rpcConn), TransportSearchServiceV1)
		delete(serviceNames, TransportSearchServiceV1)
	}
	if _, ok := serviceNames[TransportSearchServiceV2]; ok {
		services[TransportSearchServiceV2Request] = rpc.NewService(NewTransportSearchServiceV2(rpcConn), TransportSearchServiceV2)
		delete(serviceNames, TransportSearchServiceV2)
	}
	return services
}
