// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v3/accommodationv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v2/activityv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v3/activityv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v2/bookv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v2/infov2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/insurance/v1/insurancev1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1/networkv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v2/partnerv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v2/seat_mapv2grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v3/seat_mapv3grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v3/transportv3grpc"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	bookv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v2"
	infov2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v2"
	insurancev1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/insurance/v1"
	networkv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1"
	notificationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/notification/v1"
	partnerv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v2"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	seat_mapv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v3"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/handlers"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type partnerPlugin struct {
	networkv1grpc.GetNetworkFeeServiceServer
	pingv1grpc.PingServiceServer
	insurancev1grpc.InsuranceProductInfoServiceClient
	insurancev1grpc.InsuranceProductListServiceClient
	insurancev1grpc.InsuranceSearchServiceServer
	activityv2grpc.ActivitySearchServiceServer
	accommodationv3grpc.AccommodationProductInfoServiceServer
	accommodationv3grpc.AccommodationProductListServiceServer
	accommodationv3grpc.AccommodationSearchServiceServer
	partnerv2grpc.GetPartnerConfigurationServiceServer
	transportv3grpc.TransportSearchServiceServer
	transportv3grpc.TransportProductListServiceServer
	seat_mapv2grpc.SeatMapServiceServer
	seat_mapv2grpc.SeatMapAvailabilityServiceServer
	infov2grpc.CountryEntryRequirementsServiceServer
	activityv2grpc.ActivityProductInfoServiceServer
	notificationv1grpc.NotificationServiceServer
}

func (p *partnerPlugin) Validation(ctx context.Context, _ *bookv2.ValidationRequest) (*bookv2.ValidationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (Validation)", md.RequestID)

	response := bookv2.ValidationResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		ValidationId:     &typesv1.UUID{Value: md.RequestID},
		ValidationObject: nil,
		PriceDetail: &typesv2.PriceDetail{
			Price: &typesv2.Price{
				Value:    "100",
				Decimals: 0,
				Currency: &typesv2.Currency{
					Currency: &typesv2.Currency_NativeToken{},
				},
			},
		},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) ActivityProductInfo(ctx context.Context, _ *activityv3.ActivityProductInfoRequest) (*activityv3.ActivityProductInfoResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (ActivityProductInfo)", md.RequestID)

	response := activityv3.ActivityProductInfoResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Activities: []*activityv3.ActivityExtendedInfo{{
			Activity: &activityv3.Activity{
				Context:           "ActivityTest", // context
				LastModified:      timestamppb.New(time.Now()),
				ExternalSessionId: "23456", // external_session_id
				ProductCode: &typesv2.ProductCode{
					Code: "XPTFAOH15O", // supplier_code
				},
				UnitCode:    "ActivityTest", // supplier_unit_code
				ServiceCode: "TRF",          // service_code
				Bookability: &typesv1.Bookability{
					Type: typesv1.BookabilityType_BOOKABILITY_TYPE_ON_REQUEST,
					ConfirmationTime: &typesv1.Time{
						Hours:   18,
						Minutes: 0o0,
					},
				},
			},
			Units: []*activityv3.ActivityUnit{
				{
					Schedule: &typesv1.DateTimeRange{
						StartDatetime: timestamppb.New(time.Date(2024, 9, 20, 11, 0o0, 0, 0, time.UTC)), // summary.start
						EndDatetime:   timestamppb.New(time.Date(2024, 9, 20, 12, 0o0, 0, 0, time.UTC)),
					},
					Code:        "TK0001H1",                               // unit_code
					Name:        "Tuk-Tuk Sightseeing Tour (1 hour ) [1]", // unit_code_description
					Description: "starts at 11h00",                        // descriptive_text
				}, {
					Schedule: &typesv1.DateTimeRange{
						StartDatetime: timestamppb.New(time.Date(2024, 9, 20, 9, 30, 0, 0, time.UTC)), // summary.start
						EndDatetime:   timestamppb.New(time.Date(2024, 9, 20, 10, 30, 0, 0, time.UTC)),
					},
					Code:        "TK0001H0",                               // unit_code
					Name:        "Tuk-Tuk Sightseeing Tour (1 hour ) [1]", // unit_code_description
					Description: "starts at 09h30",                        // descriptive_text
				}, {
					Schedule: &typesv1.DateTimeRange{
						StartDatetime: timestamppb.New(time.Date(20024, 9, 20, 16, 30, 0, 0, time.UTC)), // summary.start
						EndDatetime:   timestamppb.New(time.Date(20024, 9, 20, 17, 30, 0, 0, time.UTC)),
					},
					Code:        "TK0001H2",                               // unit_code
					Name:        "Tuk-Tuk Sightseeing Tour (1 hour ) [2]", // unit_code_description
					Description: "starts at 16h30",                        // descriptive_text
				},
			},
			Services: []*activityv3.ActivityService{{
				Code:        "TRF",
				Name:        "incl. pickUp & dropOff",
				Description: "incl. pickUp & dropOff",
				Included: []string{
					"Exclusive English or Italian-speaking Musement guide",
					"Skip-the-line entrance to Leonardo da Vinci's Last Supper",
				},
				Excluded: []string{},
			}},
			Zones: []*activityv3.TransferZone{{
				Code: "ALT", // zone_code
				GeoTree: &typesv2.GeoTree{
					Country:      typesv2.Country_COUNTRY_PT,
					Region:       "Algarve",
					CityOrResort: "Albufeira",
				},
				PickupDropoffEvents: []*activityv3.PickupDropoffEvent{{
					LocationCode:    "AMTSPT0026",
					LocationName:    "HOTELENTRANCE / HotelEntrance",
					PickupIndicator: true,
					OtherInfo:       "HOTELENTRANCE",
					DateTime:        timestamppb.New(time.Date(20024, 9, 20, 16, 30, 0, 0, time.UTC)),
					Coordinates: &typesv2.Coordinates{
						Latitude:  37.08472,
						Longitude: -8.31469,
					},
				}},
			}},
			Descriptions: []*typesv1.LocalizedDescriptionSet{{
				Language: typesv1.Language_LANGUAGE_EN,
				Descriptions: []*typesv1.Description{{
					Category: "Tours",
					Text:     "Albufeira Tuk Tuk Experiences offers a range of exciting tours and experiences in the beautiful city of Albufeira.\n\nEmbark on a city tour aboard our comfortable and stylish Tuk Tuks. Explore the vibrant city of Albufeira in a fun and unique way with our City Tour.\n\nLearn about the Albufeira's fascinating past. Our knowledgeable and friendly guides will take you on a journey through the city's charming neighborhoods, the narrow streets of the old town, traditional architecture, local culture and iconic landmarks.\n\nIf you are a tourist visiting Albufeira or a local looking for a new perspective, our tours are designed to provide you with an immersive and memorable experience. \nGet ready to capture stunning photos and create lasting memories.\n\n- City Tour duration: \nChoose how much time you want to spend 1h, 2h or 3h.\n\nIMPORTANT NOTES:\n- Minimum 1  - Maximum 6 people.\n- Price is per vehicle and not per person.\n- Minors must be accompanied by an adult. \n- Reservations can be cancelled free of charge up to 24h before the tour starts. Less than 24h no refund. No shows are not refundable.\n\nNOT RECOMMENDED TO:\n- It\u00b4s not recommended for pregnant women and intoxicated people.\n- Not recommended to mentally or physically incapacitated people.",
				}},
			}},
			Location: &activityv3.ActivityLocation{},
			Features: []*activityv3.ActivityFeature{
				{
					Description: "Difficulty|Easy|",
					Code:        "EX_DIFFIC|EX_DIF_1",
				}, {
					Description: "What`s included|Hotel pickup and drop-off|",
					Code:        "EX_INCL|EX_INCL_HPD",
				}, {
					Description: "English, Spanish, Russian, Portuguese, Romanian",
					Code:        "Languages:",
				}, {
					Description: "We invite you to discover Albufeira  by Tuk Tuk!\nEmbark on our City Tour and explore the vibrant streets and rich history of Albufeira.\nGet ready to capture stunning photos and create lasting memories.",
					Code:        "EN Description",
				},
			},
			Tags: []*activityv3.ActivityTag{
				{
					Active: true,
					Id:     111,
					Name:   "Guided Tour",
					Slug:   "guided-tour",
				}, {
					Active: true,
					Id:     2,
					Name:   "Entrance Tickets",
					Slug:   "entrance-tickets",
				},
			},
			Languages: []typesv1.Language{
				typesv1.Language_LANGUAGE_EN,
				typesv1.Language_LANGUAGE_IT,
			},
			ContactInfo: &typesv2.ContactInfo{
				Address: []*typesv2.Address{{
					Line_1:  "Calle Sant Joan 38",
					Line_2:  "Quarter La Vileta",
					ZipCode: "07008",
					GeoTree: &typesv2.GeoTree{
						Country:      typesv2.Country_COUNTRY_ES,
						Region:       "Mallorca",
						CityOrResort: "Palma",
					},
				}},
			},
			Images: []*typesv3.Image{},
			Videos: []*typesv3.Video{{
				File: &typesv3.File{
					Name:         "Tuk Tuk Experiences",
					Url:          "video_url",
					LastModified: timestamppb.New(time.Now()),
				},
				Bitrate:   90,
				Framerate: 90,
				Category:  "Commercial",
				Width:     1920,
				Height:    1080,
				Format:    typesv3.VideoFormat_VIDEO_FORMAT_AVI,
			}},
		}},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) ActivityProductList(ctx context.Context, _ *activityv2.ActivityProductListRequest) (*activityv2.ActivityProductListResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (ActivityProductList)", md.RequestID)

	response := activityv2.ActivityProductListResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Activities: []*activityv2.Activity{{
			Context:           "ActivityTest", // context
			LastModified:      timestamppb.New(time.Now()),
			ExternalSessionId: "23456", // external_session_id
			ProductCode: &typesv2.ProductCode{
				Code: "XPTFAOH15O", // supplier_code
			},
			UnitCode:    "ActivityTest", // supplier_unit_code
			ServiceCode: "TRF",          // service_code
			Bookability: &typesv1.Bookability{
				Type: typesv1.BookabilityType_BOOKABILITY_TYPE_ON_REQUEST,
				ConfirmationTime: &typesv1.Time{
					Hours:   18,
					Minutes: 0o0,
				},
			},
		}},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) ActivitySearch(ctx context.Context, _ *activityv3.ActivitySearchRequest) (*activityv3.ActivitySearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (ActivitySearch)", md.RequestID)

	response := activityv3.ActivitySearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv3.SearchResponseMetadata{SearchId: &typesv1.UUID{Value: md.RequestID}},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) AccommodationProductInfo(ctx context.Context, _ *accommodationv3.AccommodationProductInfoRequest) (*accommodationv3.AccommodationProductInfoResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (AccommodationProductInfo)", md.RequestID)

	response := accommodationv3.AccommodationProductInfoResponse{
		Properties: []*accommodationv3.PropertyExtendedInfo{{PaymentType: "cash"}},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) AccommodationProductList(ctx context.Context, _ *accommodationv3.AccommodationProductListRequest) (*accommodationv3.AccommodationProductListResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (AccommodationProductList)", md.RequestID)

	response := accommodationv3.AccommodationProductListResponse{
		Properties: []*accommodationv3.Property{{Name: "Hotel"}},
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) AccommodationSearch(ctx context.Context, _ *accommodationv3.AccommodationSearchRequest) (*accommodationv3.AccommodationSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (AccommodationSearch)", md.RequestID)

	response := accommodationv3.AccommodationSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: md.RequestID},
		},
		Results: []*accommodationv3.AccommodationSearchResult{{
			ResultId: 0,
			QueryId:  0,
			Units: []*accommodationv3.Unit{{
				Type:             *accommodationv3.UnitType_UNIT_TYPE_ROOM.Enum(),
				SupplierRoomCode: "RMSDDB0000",
				SupplierRoomName: "Double Standard Room",
				OriginalRoomName: "Room with a view",
				TravelPeriod:     &typesv1.TravelPeriod{},
				// TravellerIds:
			}},
			TotalPriceDetail: &typesv3.PriceDetail{
				Price: &typesv3.Price{
					Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
					Value:    "199",
					Decimals: 99,
				},
				Binding:        false,
				LocallyPayable: true,
				Description:    "Off season price",
				Type: &typesv1.PriceBreakdownType{
					Code: "POS",
				},
			},
			RateRules:    []*typesv1.RateRule{{}},
			CancelPolicy: &typesv3.CancelPolicy{},
			Bookability:  &typesv1.Bookability{},
			Remarks:      "A remark",
		}},
		Travellers: []*typesv3.BasicTraveller{{
			Type:        typesv3.TravellerType(typesv1.TravelType_TRAVEL_TYPE_LEISURE),
			Birthdate:   &typesv1.Date{},
			Nationality: typesv2.Country_COUNTRY_DE,
		}},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) GetNetworkFee(ctx context.Context, request *networkv1.GetNetworkFeeRequest) (*networkv1.GetNetworkFeeResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (GetNetworkFee)", md.RequestID)

	response := networkv1.GetNetworkFeeResponse{
		NetworkFee: &networkv1.NetworkFee{
			Amount: 0,
		},
		CurrentBlockHeight: request.BlockHeight,
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) GetPartnerConfiguration(ctx context.Context, request *partnerv2.GetPartnerConfigurationRequest) (*partnerv2.GetPartnerConfigurationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (GetPartnerConfiguration)", md.RequestID)

	response := partnerv2.GetPartnerConfigurationResponse{
		PartnerConfiguration: &partnerv2.PartnerConfiguration{},
		CurrentBlockHeight:   request.GetBlockHeight(),
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) Ping(ctx context.Context, request *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (Ping)", md.RequestID)

	return &pingv1.PingResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", request.PingMessage, md.RequestID),
	}, nil
}

func (p *partnerPlugin) TransportSearch(ctx context.Context, _ *transportv3.TransportSearchRequest) (*transportv3.TransportSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TransportSearch) v3", md.RequestID)

	response := transportv3.TransportSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: md.RequestID},
		},
		ContentSourceTypes: []typesv1.ContentSourceType{
			typesv1.ContentSourceType_CONTENT_SOURCE_TYPE_GDS,
			typesv1.ContentSourceType_CONTENT_SOURCE_TYPE_NDC,
		},
		Results: []*transportv3.TransportSearchResult{{
			ResultId:     0,
			QueryId:      0,
			OfferId:      "123456",
			TravellerIds: []int32{123, 456},
			TravellingTrips: []*transportv3.TripExtended{{
				SupplierCode: &typesv2.SupplierProductCode{
					SupplierCode:   "XY",
					SupplierNumber: 123,
				},
				Baggage: &typesv1.Baggage{
					MaxCount: 5,
					MaxWeight: &typesv1.Weight{
						Value: 20,
						Unit:  typesv1.WeightUnit_WEIGHT_UNIT_KILOGRAM,
					},
					MaxDimension: &typesv1.Dimension{
						Length: 45,
						Width:  90,
						Height: 75,
						Unit:   typesv1.LengthUnit_LENGTH_UNIT_CENTIMETER,
					},
				},
				Price: &typesv3.Price{
					Value:    "180",
					Decimals: 2,
					Currency: &typesv3.Currency{
						Currency: &typesv3.Currency_IsoCurrency{
							IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
						},
					},
				},
				Segments: []*transportv3.SegmentExtended{{
					Info: &transportv3.Segment{
						SegmentId:    "SEG123",
						ProviderCode: "BAW", // British Airways
						ProductCode: &typesv2.ProductCode{
							Code:   "BA",
							Number: 123,
							Type:   typesv2.ProductCodeType_PRODUCT_CODE_TYPE_SUPPLIER,
						},
						SupplierCode: &typesv2.SupplierProductCode{
							SupplierCode:   "BAW",
							SupplierNumber: 123,
						},
						Departure: &transportv3.TransitEvent{
							DateTime: timestamppb.New(time.Date(2024, 9, 20, 9, 0, 0, 0, time.UTC)),
							LocationCode: &typesv2.LocationCode{
								Code: "LHR",
								Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_ICAO_CODE,
							},
						},
						Arrival: &transportv3.TransitEvent{
							DateTime: timestamppb.New(time.Date(2024, 9, 20, 11, 55, 0, 0, time.UTC)),
							LocationCode: &typesv2.LocationCode{
								Code: "JFK",
								Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_ICAO_CODE,
							},
						},
						SegmentDuration: &typesv1.Duration{
							Minutes: 475,
						},
						SegmentDistance: &typesv1.Length{
							Value: 5539,
							Unit:  typesv1.LengthUnit_LENGTH_UNIT_KILOMETER,
						},
					},
					ServiceTypeCode:        "Y",
					ServiceTypeDescription: "Economy Class",
					Services: []*typesv3.ServiceFact{{
						Code:        "EC",
						Description: "Early Check-in with Priority Boarding",
						PriceDetail: &typesv3.PriceDetail{
							Price: &typesv3.Price{
								Value:    "10",
								Decimals: 2,
								Currency: &typesv3.Currency{
									Currency: &typesv3.Currency_IsoCurrency{
										IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
									},
								},
							},
							Binding:        false,
							Description:    "Early Check-in with Priority Boarding",
							LocallyPayable: true,
						},
						AvailabilityType: typesv3.ServiceAvailabilityType_SERVICE_AVAILABILITY_TYPE_OPTIONAL,
						ChargeBasis:      typesv3.ChargeBasisType_CHARGE_BASIS_TYPE_PER_PAX,
						Quantity:         1,
						Details: []*typesv3.ServiceFact{
							{
								Code:             "PB",
								Description:      "Priority Boarding",
								AvailabilityType: typesv3.ServiceAvailabilityType_SERVICE_AVAILABILITY_TYPE_INCLUDED,
								ChargeBasis:      typesv3.ChargeBasisType_CHARGE_BASIS_TYPE_PER_PAX,
							}, {
								Code:             "HB",
								Description:      "Hand Baggage 10kg",
								AvailabilityType: typesv3.ServiceAvailabilityType_SERVICE_AVAILABILITY_TYPE_INCLUDED,
								ChargeBasis:      typesv3.ChargeBasisType_CHARGE_BASIS_TYPE_PER_PAX,
								Quantity:         1,
							},
						},
					}},
					MinPax: 1,
					MaxPax: 3,
				}},
			}},
			TotalPrice: &typesv3.PriceDetail{
				Price: &typesv3.Price{
					Value:    "180",
					Decimals: 2,
					Currency: &typesv3.Currency{
						Currency: &typesv3.Currency_IsoCurrency{
							IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
						},
					},
				},
				Binding:     true,
				Description: "Total price including all taxes and fees",
				Breakdowns: []*typesv3.PriceDetail{
					{
						Price: &typesv3.Price{
							Value:    "150",
							Decimals: 2,
							Currency: &typesv3.Currency{
								Currency: &typesv3.Currency_IsoCurrency{
									IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
								},
							},
						},
						Description: "Base price",
						Type: &typesv1.PriceBreakdownType{
							Code: "BASE",
							Type: &typesv1.PriceBreakdownType_PriceType{
								PriceType: typesv1.PriceType_PRICE_TYPE_BASE_RATE,
							},
						},
					}, {
						Price: &typesv3.Price{
							Value:    "30",
							Decimals: 2,
							Currency: &typesv3.Currency{
								Currency: &typesv3.Currency_IsoCurrency{
									IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
								},
							},
						},
						Description: "VAT",
						Type: &typesv1.PriceBreakdownType{
							Code: "VAT",
							Type: &typesv1.PriceBreakdownType_TaxCode{
								TaxCode: typesv1.TaxCode_TAX_CODE_VALUE_ADDED_TAX,
							},
						},
					},
				},
			},
			RateRules: []*typesv1.RateRule{{
				RateType:        typesv1.RateRuleType_RATE_RULE_TYPE_NON_REFUNDABLE,
				RateDescription: "Non-refundable fare",
			}},
			Links: []*typesv1.Link{{
				Type:        typesv1.LinkType_LINK_TYPE_BOOKING,
				Description: "Booking link",
				Ref:         "https://api.example.com/bookings/123456",
			}},
			Bookability: &typesv1.Bookability{
				Type: typesv1.BookabilityType_BOOKABILITY_TYPE_AVAILABLE,
				ConfirmationTime: &typesv1.Time{
					Hours:   1,
					Minutes: 0,
				},
			},
			Validity: &typesv1.DateTimeRange{
				StartDatetime: &timestamppb.Timestamp{
					Seconds: time.Now().Unix(),
				},
				EndDatetime: &timestamppb.Timestamp{
					Seconds: time.Now().Add(24 * time.Hour).Unix(),
				},
			},
			CancelPolicy: &typesv3.CancelPolicy{
				Refundable: true,
				FreeCancellationUpto: &timestamppb.Timestamp{
					Seconds: time.Now().Add(48 * time.Hour).Unix(),
				},
				CancelPenalties: []*typesv3.CancelPenalty{{
					DatetimeRange: &typesv1.DateTimeRange{
						StartDatetime: &timestamppb.Timestamp{
							Seconds: time.Now().Add(48 * time.Hour).Unix(),
						},
						EndDatetime: &timestamppb.Timestamp{
							Seconds: time.Now().Add(72 * time.Hour).Unix(),
						},
					},
					Value: &typesv3.Price{
						Value:    "50",
						Decimals: 2,
						Currency: &typesv3.Currency{
							Currency: &typesv3.Currency_IsoCurrency{
								IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
							},
						},
					},
					Description: "Late cancellation fee",
				}},
			},
			ChangePolicy: &typesv3.ChangePolicy{
				ChangeAllowed: true,
				FreeChangeUpto: &timestamppb.Timestamp{
					Seconds: time.Now().Add(48 * time.Hour).Unix(),
				},
				ChangeTypes: []*typesv3.ChangeType{{
					Code: "CHANGE",
					DatetimeRange: &typesv1.DateTimeRange{
						StartDatetime: &timestamppb.Timestamp{
							Seconds: time.Now().Add(48 * time.Hour).Unix(),
						},
						EndDatetime: &timestamppb.Timestamp{
							Seconds: time.Now().Add(72 * time.Hour).Unix(),
						},
					},
					Value: &typesv3.Price{
						Value:    "30",
						Decimals: 2,
						Currency: &typesv3.Currency{
							Currency: &typesv3.Currency_IsoCurrency{
								IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
							},
						},
					},
					Description: "Late change fee",
				}},
			},
			Observations: "Example transport search result with all fields populated",
		}},
		Travellers: []*typesv3.BasicTraveller{
			{
				TravellerId: 123,
				Type:        typesv3.TravellerType_TRAVELLER_TYPE_ADULT,
			}, {
				TravellerId: 456,
				Type:        typesv3.TravellerType_TRAVELLER_TYPE_CHILD,
			},
		},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) TransportProductList(ctx context.Context, _ *transportv3.TransportProductListRequest) (*transportv3.TransportProductListResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TransportProductList)", md.RequestID)

	response := transportv3.TransportProductListResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Trips: []*transportv3.TripBasic{{
			SupplierCode: &typesv2.SupplierProductCode{
				SupplierCode:   "XY",
				SupplierNumber: 123,
			},
			Segments: []*transportv3.Segment{{
				SegmentId:    "SEG123",
				ProviderCode: "I2",
				ProductCode: &typesv2.ProductCode{
					Code:   "XY",
					Number: 123,
					Type:   typesv2.ProductCodeType_PRODUCT_CODE_TYPE_SUPPLIER,
				},
				SupplierCode: &typesv2.SupplierProductCode{
					SupplierCode:   "XY",
					SupplierNumber: 123,
				},
				Departure: &transportv3.TransitEvent{
					DateTime: timestamppb.New(time.Date(2024, 9, 20, 9, 0, 0, 0, time.UTC)),
					LocationCode: &typesv2.LocationCode{
						Code: "LHR",
						Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_ICAO_CODE,
					},
				},
				Arrival: &transportv3.TransitEvent{
					DateTime: timestamppb.New(time.Date(2024, 9, 20, 11, 55, 0, 0, time.UTC)),
					LocationCode: &typesv2.LocationCode{
						Code: "JFK",
						Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_ICAO_CODE,
					},
				},
				SegmentDuration: &typesv1.Duration{
					Minutes: 475,
				},
				SegmentDistance: &typesv1.Length{
					Value: 5539,
					Unit:  typesv1.LengthUnit_LENGTH_UNIT_KILOMETER,
				},
			}},
		}, {
			SupplierCode: &typesv2.SupplierProductCode{
				SupplierCode:   "XY",
				SupplierNumber: 456,
			},
			Segments: []*transportv3.Segment{{
				SegmentId:    "SEG456",
				ProviderCode: "I2",
				ProductCode: &typesv2.ProductCode{
					Code:   "XY",
					Number: 456,
					Type:   typesv2.ProductCodeType_PRODUCT_CODE_TYPE_SUPPLIER,
				},
				SupplierCode: &typesv2.SupplierProductCode{
					SupplierCode:   "XY",
					SupplierNumber: 456,
				},
				Departure: &transportv3.TransitEvent{
					DateTime: timestamppb.New(time.Date(2024, 9, 27, 23, 55, 0, 0, time.UTC)),
					LocationCode: &typesv2.LocationCode{
						Code: "SIN",
						Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_ICAO_CODE,
					},
				},
				Arrival: &transportv3.TransitEvent{
					DateTime: timestamppb.New(time.Date(2024, 9, 28, 9, 30, 0, 0, time.UTC)),
					LocationCode: &typesv2.LocationCode{
						Code: "MEL",
						Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_ICAO_CODE,
					},
				},
				SegmentDuration: &typesv1.Duration{
					Minutes: 455,
				},
				SegmentDistance: &typesv1.Length{
					Value: 6024,
					Unit:  typesv1.LengthUnit_LENGTH_UNIT_KILOMETER,
				},
			}},
		}, {
			SupplierCode: &typesv2.SupplierProductCode{
				SupplierCode:   "XY",
				SupplierNumber: 789,
			},
			Segments: []*transportv3.Segment{{
				SegmentId:    "SEG789",
				ProviderCode: "DB",
				ProductCode: &typesv2.ProductCode{
					Code:   "XY",
					Number: 789,
					Type:   typesv2.ProductCodeType_PRODUCT_CODE_TYPE_SUPPLIER,
				},
				SupplierCode: &typesv2.SupplierProductCode{
					SupplierCode:   "XY",
					SupplierNumber: 789,
				},
				Departure: &transportv3.TransitEvent{
					DateTime: timestamppb.New(time.Date(2024, 9, 27, 15, 30, 0, 0, time.UTC)),
					LocationCode: &typesv2.LocationCode{
						Code: "PAR",
						Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_UNSPECIFIED,
					},
				},
				Arrival: &transportv3.TransitEvent{
					DateTime: timestamppb.New(time.Date(2024, 9, 28, 16, 47, 0, 0, time.UTC)),
					LocationCode: &typesv2.LocationCode{
						Code: "LON",
						Type: typesv2.LocationCodeType_LOCATION_CODE_TYPE_UNSPECIFIED,
					},
				},
				SegmentDuration: &typesv1.Duration{
					Minutes: 137,
				},
				SegmentDistance: &typesv1.Length{
					Value: 495,
					Unit:  typesv1.LengthUnit_LENGTH_UNIT_KILOMETER,
				},
			}},
		}},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) SeatMap(ctx context.Context, _ *seat_mapv3.SeatMapRequest) (*seat_mapv3.SeatMapResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (SeatMap)", md.RequestID)

	response := seat_mapv3.SeatMapResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		SeatMap: &typesv3.SeatMap{
			Id: md.RequestID,
			Sections: []*typesv3.Section{
				{
					Id: "123ST",
					Names: []*typesv1.LocalizedString{
						{
							Language: typesv1.Language_LANGUAGE_EN,
							Text:     "North Stand",
						}, {
							Language: typesv1.Language_LANGUAGE_DE,
							Text:     "Nordtribüne",
						},
					},
					SeatInfo: &typesv3.Section_SeatList{
						SeatList: &typesv3.SeatList{
							Seats: []*typesv3.Seat{
								{
									Id: "1A",
									Location: &typesv3.SeatLocation{
										Location: &typesv3.SeatLocation_Vector{
											Vector: &typesv3.VectorSeatLocation{
												Label: "section-North-Stand-26-34-2-label",
											},
										},
									},
								}, {
									Id: "2A",
									Location: &typesv3.SeatLocation{
										Location: &typesv3.SeatLocation_Vector{
											Vector: &typesv3.VectorSeatLocation{
												Label: "section-North-Stand-26-34-2-label",
											},
										},
									},
									Restrictions: []*typesv3.LocalizedSeatAttributeSet{{
										Language: typesv1.Language_LANGUAGE_EN,
										SeatAttributes: []*typesv3.SeatAttribute{{
											Name:        "Restricted Vision",
											Description: "Seat behind a column",
										}},
									}},
									Features: []*typesv3.LocalizedSeatAttributeSet{
										{
											Language: typesv1.Language_LANGUAGE_EN,
											SeatAttributes: []*typesv3.SeatAttribute{{
												Name:        "Discount",
												Description: "Discount due to restricted vision up to 80%",
												Value:       int32(80),
											}},
										}, {
											Language: typesv1.Language_LANGUAGE_DE,
											SeatAttributes: []*typesv3.SeatAttribute{{
												Name:        "Rabatt",
												Description: "Hinter der Säule - bis zu 80% Rabatt",
												Value:       int32(80),
											}},
										},
									},
								},
							},
						},
					},
					Image: &typesv3.Image{
						File: &typesv3.File{
							Name:         "String",
							Url:          "https://camino.network/static/images/6HibYS9gzR-1800.webp", // TODO: replace with an actual image
							LastModified: timestamppb.New(time.Now()),
						},
						Width:  50,
						Height: 50,
					},
					LocalizedDescriptions: []*typesv1.LocalizedDescriptionSet{{
						Language: typesv1.Language_LANGUAGE_EN,
						Descriptions: []*typesv1.Description{{
							Category: "General",
							Text:     "Leather Seats",
						}},
					}},
				}, {
					Id: "124ST",
					Names: []*typesv1.LocalizedString{
						{
							Language: typesv1.Language_LANGUAGE_EN,
							Text:     "East Stand",
						}, {
							Language: typesv1.Language_LANGUAGE_DE,
							Text:     "Osttribüne",
						},
					},
					SeatInfo: &typesv3.Section_SeatList{
						SeatList: &typesv3.SeatList{
							Seats: []*typesv3.Seat{
								{
									Id: "31F",
									Location: &typesv3.SeatLocation{
										Location: &typesv3.SeatLocation_Vector{
											Vector: &typesv3.VectorSeatLocation{
												Label: "section-East-Stand-26-34-2-label",
											},
										},
									},
								}, {
									Id: "32F",
									Location: &typesv3.SeatLocation{
										Location: &typesv3.SeatLocation_Vector{
											Vector: &typesv3.VectorSeatLocation{
												Label: "section-East-Stand-26-34-2-label",
											},
										},
									},
								},
							},
						},
					},
					Image: &typesv3.Image{
						File: &typesv3.File{
							Name:         "String",
							Url:          "https://camino.network/static/images/6HibYS9gzR-1800.webp",
							LastModified: timestamppb.New(time.Now()),
						},
						Width:  50,
						Height: 50,
					},
					LocalizedDescriptions: []*typesv1.LocalizedDescriptionSet{
						{
							Language: typesv1.Language_LANGUAGE_EN,
							Descriptions: []*typesv1.Description{{
								Category: "General",
								Text:     "Seats",
							}},
						}, {
							Language: typesv1.Language_LANGUAGE_DE,
							Descriptions: []*typesv1.Description{{
								Category: "Allgemein",
								Text:     "Sitz",
							}},
						},
					},
				},
			},
		},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) SeatMapAvailability(ctx context.Context, _ *seat_mapv3.SeatMapAvailabilityRequest) (*seat_mapv3.SeatMapAvailabilityResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (SeatMapAvailability)", md.RequestID)

	response := seat_mapv3.SeatMapAvailabilityResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		SeatMap: &typesv3.SeatMapInventory{
			Id: "123ST",
			Sections: []*typesv3.SectionInventory{
				{
					Id: "A",
					SeatInfo: &typesv3.SectionInventory_SeatList{
						SeatList: &typesv3.SeatInventory{
							Ids: []string{"1A", "1B"},
						},
					},
				}, {
					Id:       "B",
					SeatInfo: &typesv3.SectionInventory_SeatCount{SeatCount: &wrapperspb.Int32Value{Value: 32}},
				},
			},
		},
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) CountryEntryRequirements(ctx context.Context, _ *infov2.CountryEntryRequirementsRequest) (*infov2.CountryEntryRequirementsResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (CountryEntryRequirements)", md.RequestID)

	response := infov2.CountryEntryRequirementsResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Categories: []*infov2.CountryEntryRequirementCategory{{
			Key: "entry",
			Names: []*typesv1.LocalizedString{
				{
					Text:     "Entry",
					Language: typesv1.Language_LANGUAGE_EN,
				}, {
					Text:     "Einreise",
					Language: typesv1.Language_LANGUAGE_DE,
				},
			},
			SubCategories: []*infov2.CountryEntryRequirementCategory{{
				Key: "entry_documents",
				Names: []*typesv1.LocalizedString{
					{
						Text:     "Required entry forms and documents",
						Language: typesv1.Language_LANGUAGE_EN,
					}, {
						Text:     "Erforderliche Formulare und Dokumente für die Einreise",
						Language: typesv1.Language_LANGUAGE_DE,
					},
				},
				Items: []*infov2.CountryEntryRequirementItem{{
					Key: "ErVisaText",
					Info: []*infov2.LocalizedItemInfo{
						{
							Name:        "Visa required for stay",
							Description: "<div><p>A visa is required for the stay. This can be applied for as an e-Visa or on arrival as a \"Visa on Arrival\". </p></div><div><div>Travellers with eVisa are permitted to stay up to 30 days in Egypt.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visa fees</a></p><div>Visa fees must be paid in cash in euros or US dollars.</div>",
							Language:    typesv1.Language_LANGUAGE_EN,
						}, {
							Name:        "Visum erforderlich für Aufenthalt",
							Description: "<div><p>Es ist ein Visum für den Aufenthalt erforderlich. Dieses kann als e-Visum oder bei Ankunft als \"Visa on Arrival\" beantragt werden. </p></div><div><div>Reisende mit eVisa dürfen sich bis zu 30 Tage im Land aufhalten.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visumgebühren</a></p><div>Die Visumgebühren sind in Euro oder US-Dollar bar zu zahlen.</div>",
							Language:    typesv1.Language_LANGUAGE_DE,
						},
					},
					LastSignificantUpdate: timestamppb.New(time.Now()),
					Status:                infov2.ItemStatus_ITEM_STATUS_TRUE,
				}},
			}},
		}},
		Items: []*infov2.CountryEntryRequirementItem{
			{
				Key: "EntryDocumentsRequired",
				Info: []*infov2.LocalizedItemInfo{
					{
						Name:        "Entry forms",
						Description: "<div><p>Individuals must fill out a <a href=\"https://www.egyptair.com/en/about-egyptair/news-and-press/Documents/%D8%A7%D9%84%D8%A7%D9%95%D9%82%D8%B1%D8%A7%D8%B1%20%D8%A7%D9%84%D8%B5%D8%AD%D9%8A%20%D9%84%D8%BA%D9%8A%D8%B1%20%D8%A7%D9%84%D9%85%D8%B5%D8%B1%D9%8A%D9%8A%D9%86%20%28%D8%A7%D9%84%D8%A7%D9%94%D8%AC%D8%A7%D9%86%D8%A8%29.pdf\" rel=\"noopener noreferrer\" target=\"_blank\">health form</a> upon entry, which they can complete either at the airport, on the plane, or beforehand.</p></div>",
						Language:    typesv1.Language_LANGUAGE_EN,
					}, {
						Name:        "Einreiseformulare",
						Description: "<div><div><p>Personen müssen bei Einreise ein <a href=\"https://www.egyptair.com/en/about-egyptair/news-and-press/Documents/%D8%A7%D9%84%D8%A7%D9%95%D9%82%D8%B1%D8%A7%D8%B1%20%D8%A7%D9%84%D8%B5%D8%AD%D9%8A%20%D9%84%D8%BA%D9%8A%D8%B1%20%D8%A7%D9%84%D9%85%D8%B5%D8%B1%D9%8A%D9%8A%D9%86%20%28%D8%A7%D9%84%D8%A7%D9%94%D8%AC%D8%A7%D9%86%D8%A8%29.pdf\" rel=\"noopener noreferrer\" target=\"_blank\">Gesundheitsformular</a> abgeben, welches entweder am Flughafen, im Flugzeug oder vor Antritt der Reise ausfüllen.</p></div></div>",
						Language:    typesv1.Language_LANGUAGE_DE,
					},
				},
				LastSignificantUpdate: timestamppb.New(time.Now()),
				Status:                infov2.ItemStatus_ITEM_STATUS_FALSE,
			}, {
				Key: "ErVisaText",
				Info: []*infov2.LocalizedItemInfo{
					{
						Name:        "Visa required for stay",
						Description: "<div><p>A visa is required for the stay. This can be applied for as an e-Visa or on arrival as a \"Visa on Arrival\". </p></div><div><div>Travellers with eVisa are permitted to stay up to 30 days in Egypt.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visa fees</a></p><div>Visa fees must be paid in cash in euros or US dollars.</div>",
						Language:    typesv1.Language_LANGUAGE_EN,
					}, {
						Name:        "Visum erforderlich für Aufenthalt",
						Description: "<div><p>Es ist ein Visum für den Aufenthalt erforderlich. Dieses kann als e-Visum oder bei Ankunft als \"Visa on Arrival\" beantragt werden. </p></div><div><div>Reisende mit eVisa dürfen sich bis zu 30 Tage im Land aufhalten.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visumgebühren</a></p><div>Die Visumgebühren sind in Euro oder US-Dollar bar zu zahlen.</div>",
						Language:    typesv1.Language_LANGUAGE_DE,
					},
				},
				LastSignificantUpdate: timestamppb.New(time.Now()),
				Status:                infov2.ItemStatus_ITEM_STATUS_TRUE,
			},
		},
	}
	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) InsuranceProductInfo(ctx context.Context, _ *insurancev1.InsuranceProductInfoRequest) (*insurancev1.InsuranceProductInfoResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (InsuranceProductInfo)", md.RequestID)

	response := insurancev1.InsuranceProductInfoResponse{
		// TODO: add an example
	}

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) InsuranceProductList(ctx context.Context, _ *insurancev1.InsuranceProductListRequest) (*insurancev1.InsuranceProductListResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (InsuranceProductList)", md.RequestID)

	response := insurancev1.InsuranceProductListResponse{
		// TODO: add an example
	}

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) InsuranceSearch(ctx context.Context, _ *insurancev1.InsuranceSearchRequest) (*insurancev1.InsuranceSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (InsuranceSearch)", md.RequestID)

	response := insurancev1.InsuranceSearchResponse{
		// TODO: add an example
	}

	return &response, grpc.SendHeader(ctx, md.ToGrpcMD())
}

func (p *partnerPlugin) TokenBoughtNotification(ctx context.Context, _ *notificationv1.TokenBought) (*emptypb.Empty, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TokenBoughtNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}

func (p *partnerPlugin) TokenExpiredNotification(ctx context.Context, _ *notificationv1.TokenExpired) (*emptypb.Empty, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s (TokenExpiredNotification)", md.RequestID)

	return &emptypb.Empty{}, nil
}

func main() {
	grpcServer := grpc.NewServer()

	networkv1grpc.RegisterGetNetworkFeeServiceServer(grpcServer, &partnerPlugin{})
	pingv1grpc.RegisterPingServiceServer(grpcServer, &partnerPlugin{})
	insurancev1grpc.RegisterInsuranceProductInfoServiceServer(grpcServer, &partnerPlugin{})
	insurancev1grpc.RegisterInsuranceProductListServiceServer(grpcServer, &partnerPlugin{})
	insurancev1grpc.RegisterInsuranceSearchServiceServer(grpcServer, &partnerPlugin{})
	activityv3grpc.RegisterActivityProductInfoServiceServer(grpcServer, &partnerPlugin{})
	activityv3grpc.RegisterActivitySearchServiceServer(grpcServer, &partnerPlugin{})
	accommodationv3grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &partnerPlugin{})
	accommodationv3grpc.RegisterAccommodationProductListServiceServer(grpcServer, &partnerPlugin{})
	accommodationv3grpc.RegisterAccommodationSearchServiceServer(grpcServer, &partnerPlugin{})
	partnerv2grpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, &partnerPlugin{})
	bookv2grpc.RegisterValidationServiceServer(grpcServer, &partnerPlugin{})
	transportv3grpc.RegisterTransportProductListServiceServer(grpcServer, &partnerPlugin{})
	transportv3grpc.RegisterTransportSearchServiceServer(grpcServer, &partnerPlugin{})
	seat_mapv3grpc.RegisterSeatMapServiceServer(grpcServer, &partnerPlugin{})
	seat_mapv3grpc.RegisterSeatMapAvailabilityServiceServer(grpcServer, &partnerPlugin{})
	infov2grpc.RegisterCountryEntryRequirementsServiceServer(grpcServer, &partnerPlugin{})
	notificationv1grpc.RegisterNotificationServiceServer(grpcServer, &partnerPlugin{})

	bookv2grpc.RegisterMintServiceServer(grpcServer, &handlers.MintServiceV2Server{})
	bookv1grpc.RegisterMintServiceServer(grpcServer, &handlers.MintServiceV1Server{})

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
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
