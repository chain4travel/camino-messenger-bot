package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1/activityv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1/bookv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v1/infov1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1/networkv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1/partnerv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1/pingv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v1/seat_mapv1grpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1/transportv1grpc"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	bookv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1"
	infov1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v1"
	networkv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1"
	partnerv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1"
	pingv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1"
	seat_mapv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v1"
	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
)

type partnerPlugin struct {
	activityv1grpc.ActivitySearchServiceServer
	accommodationv1grpc.AccommodationProductInfoServiceServer
	accommodationv1grpc.AccommodationProductListServiceServer
	accommodationv1grpc.AccommodationSearchServiceServer
	networkv1grpc.GetNetworkFeeServiceServer
	partnerv1grpc.GetPartnerConfigurationServiceServer
	pingv1grpc.PingServiceServer
	transportv1grpc.TransportSearchServiceServer
	seat_mapv1grpc.SeatMapServiceServer
	seat_mapv1grpc.SeatMapAvailabilityServiceServer
	infov1grpc.CountryEntryRequirementsServiceServer
	activityv1grpc.ActivityProductInfoServiceServer
}

func (p *partnerPlugin) Mint(ctx context.Context, _ *bookv1.MintRequest) (*bookv1.MintResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := bookv1.MintResponse{
		MintId: &typesv1.UUID{Value: md.RequestID},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		Price: &typesv1.Price{
			Value:    "1",
			Decimals: 9,
		},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) Validation(ctx context.Context, _ *bookv1.ValidationRequest) (*bookv1.ValidationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := bookv1.ValidationResponse{
		Header:           nil,
		ValidationId:     &typesv1.UUID{Value: md.RequestID},
		ValidationObject: nil,
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) ActivitySearch(ctx context.Context, _ *activityv1.ActivitySearchRequest) (*activityv1.ActivitySearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := activityv1.ActivitySearchResponse{
		Header:   nil,
		Metadata: &typesv1.SearchResponseMetadata{SearchId: &typesv1.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) AccommodationProductInfo(ctx context.Context, _ *accommodationv1.AccommodationProductInfoRequest) (*accommodationv1.AccommodationProductInfoResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := accommodationv1.AccommodationProductInfoResponse{
		Properties: []*accommodationv1.PropertyExtendedInfo{{PaymentType: "cash"}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) AccommodationProductList(ctx context.Context, _ *accommodationv1.AccommodationProductListRequest) (*accommodationv1.AccommodationProductListResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := accommodationv1.AccommodationProductListResponse{
		Properties: []*accommodationv1.Property{{Name: "Hotel"}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) AccommodationSearch(ctx context.Context, _ *accommodationv1.AccommodationSearchRequest) (*accommodationv1.AccommodationSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := accommodationv1.AccommodationSearchResponse{
		Header:   nil,
		Metadata: &typesv1.SearchResponseMetadata{SearchId: &typesv1.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) GetNetworkFee(ctx context.Context, request *networkv1.GetNetworkFeeRequest) (*networkv1.GetNetworkFeeResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := networkv1.GetNetworkFeeResponse{
		NetworkFee: &networkv1.NetworkFee{
			Amount: 0,
		},
		CurrentBlockHeight: request.BlockHeight,
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) GetPartnerConfiguration(ctx context.Context, request *partnerv1.GetPartnerConfigurationRequest) (*partnerv1.GetPartnerConfigurationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := partnerv1.GetPartnerConfigurationResponse{
		PartnerConfiguration: &partnerv1.PartnerConfiguration{},
		CurrentBlockHeight:   request.GetBlockHeight(),
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) Ping(ctx context.Context, request *pingv1.PingRequest) (*pingv1.PingResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	return &pingv1.PingResponse{
		Header:      nil,
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", request.PingMessage, md.RequestID),
	}, nil
}

func (p *partnerPlugin) TransportSearch(ctx context.Context, _ *transportv1.TransportSearchRequest) (*transportv1.TransportSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := transportv1.TransportSearchResponse{
		Header:   nil,
		Metadata: &typesv1.SearchResponseMetadata{SearchId: &typesv1.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}
func (p *partnerPlugin) SeatMap(ctx context.Context, request *seat_mapv1.SeatMapRequest) (*seat_mapv1.SeatMapResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := seat_mapv1.SeatMapResponse{
		Header: nil,
		SeatMap: &typesv1.SeatMap{
			Id: md.RequestID,
			Sections: []*typesv1.Section{{
				Id: "123ST",
				Names: []*typesv1.LocalizedString{{
					Language: typesv1.Language_LANGUAGE_EN,
					Text:     "North Stand",
				},
					{
						Language: typesv1.Language_LANGUAGE_DE,
						Text:     "Nordtribüne",
					},
				},
				SeatInfo: &typesv1.Section_SeatList{
					SeatList: &typesv1.SeatList{
						Seats: []*typesv1.Seat{
							{
								Id: "1A",
								Location: &typesv1.SeatLocation{
									Location: &typesv1.SeatLocation_Vector{
										Vector: &typesv1.VectorSeatLocation{
											Label: "section-North-Stand-26-34-2-label",
										},
									},
								},
							},
							{
								Id: "2A",
								Location: &typesv1.SeatLocation{
									Location: &typesv1.SeatLocation_Vector{
										Vector: &typesv1.VectorSeatLocation{
											Label: "section-North-Stand-26-34-2-label",
										},
									},
								},
								Restrictions: []*typesv1.LocalizedSeatAttributeSet{
									{
										Language: typesv1.Language_LANGUAGE_EN,
										SeatAttributes: []*typesv1.SeatAttribute{
											{
												Name:        "Restricted Vision",
												Description: "Seat behind a column",
											},
										},
									},
								},
								Features: []*typesv1.LocalizedSeatAttributeSet{
									{
										Language: typesv1.Language_LANGUAGE_EN,
										SeatAttributes: []*typesv1.SeatAttribute{
											{
												Name:        "Discount",
												Description: "Discount due to restricted vision up to 80%",
												Value:       int32(80),
											},
										},
									},
									{
										Language: typesv1.Language_LANGUAGE_DE,
										SeatAttributes: []*typesv1.SeatAttribute{
											{
												Name:        "Rabatt",
												Description: "Hinter der Säule - bis zu 80% Rabatt",
												Value:       int32(80),
											},
										},
									},
								},
							},
						},
					},
				},
				Image: &typesv1.Image{
					File: &typesv1.File{
						Name:         "String",
						Url:          "https://camino.network/static/images/6HibYS9gzR-1800.webp", //TODO: replace with an actual image
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
				},
				},
			},
				{
					Id: "124ST",
					Names: []*typesv1.LocalizedString{{
						Language: typesv1.Language_LANGUAGE_EN,
						Text:     "East Stand",
					}, {
						Language: typesv1.Language_LANGUAGE_DE,
						Text:     "Osttribüne",
					}},
					SeatInfo: &typesv1.Section_SeatList{
						SeatList: &typesv1.SeatList{
							Seats: []*typesv1.Seat{
								{
									Id: "31F",
									Location: &typesv1.SeatLocation{
										Location: &typesv1.SeatLocation_Vector{
											Vector: &typesv1.VectorSeatLocation{
												Label: "section-East-Stand-26-34-2-label",
											},
										},
									},
								},
								{
									Id: "32F",
									Location: &typesv1.SeatLocation{
										Location: &typesv1.SeatLocation_Vector{
											Vector: &typesv1.VectorSeatLocation{
												Label: "section-East-Stand-26-34-2-label",
											},
										},
									},
								},
							},
						},
					},
					Image: &typesv1.Image{
						File: &typesv1.File{
							Name:         "String",
							Url:          "https://camino.network/static/images/6HibYS9gzR-1800.webp",
							LastModified: timestamppb.New(time.Now()),
						},
						Width:  50,
						Height: 50,
					},
					LocalizedDescriptions: []*typesv1.LocalizedDescriptionSet{{
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
			}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}
func (p *partnerPlugin) SeatMapAvailability(ctx context.Context, request *seat_mapv1.SeatMapAvailabilityRequest) (*seat_mapv1.SeatMapAvailabilityResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := seat_mapv1.SeatMapAvailabilityResponse{
		Header: nil,
		SeatMap: &typesv1.SeatMapInventory{
			Id: "123ST",
			Sections: []*typesv1.SectionInventory{
				{
					Id: "A",
					SeatInfo: &typesv1.SectionInventory_SeatList{
						SeatList: &typesv1.SeatInventory{
							Ids: []string{"1A", "1B"},
						},
					}},
				{
					Id:       "B",
					SeatInfo: &typesv1.SectionInventory_SeatCount{SeatCount: &wrapperspb.Int32Value{Value: 32}},
				}},
		},
	}

	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) CountryEntryRequirements(ctx context.Context, request *infov1.CountryEntryRequirementsRequest) (*infov1.CountryEntryRequirementsResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	grpc.SendHeader(ctx, md.ToGrpcMD())
	response := infov1.CountryEntryRequirementsResponse{
		Header: nil,
		Categories: []*infov1.CountryEntryRequirementCategory{{
			Key: "entry",
			Names: []*typesv1.LocalizedString{{
				Text:     "Entry",
				Language: typesv1.Language_LANGUAGE_EN,
			}, {
				Text:     "Einreise",
				Language: typesv1.Language_LANGUAGE_DE,
			}},
			SubCategories: []*infov1.CountryEntryRequirementCategory{{
				Key: "entry_documents",
				Names: []*typesv1.LocalizedString{{
					Text:     "Required entry forms and documents",
					Language: typesv1.Language_LANGUAGE_EN,
				}, {
					Text:     "Erforderliche Formulare und Dokumente für die Einreise",
					Language: typesv1.Language_LANGUAGE_DE,
				}},
				Items: []*infov1.CountryEntryRequirementItem{{
					Key: "ErVisaText",
					Info: []*infov1.LocalizedItemInfo{{
						Name:        "Visa required for stay",
						Description: "<div><p>A visa is required for the stay. This can be applied for as an e-Visa or on arrival as a \"Visa on Arrival\". </p></div><div><div>Travellers with eVisa are permitted to stay up to 30 days in Egypt.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visa fees</a></p><div>Visa fees must be paid in cash in euros or US dollars.</div>",
						Language:    typesv1.Language_LANGUAGE_EN,
					}, {
						Name:        "Visum erforderlich für Aufenthalt",
						Description: "<div><p>Es ist ein Visum für den Aufenthalt erforderlich. Dieses kann als e-Visum oder bei Ankunft als \"Visa on Arrival\" beantragt werden. </p></div><div><div>Reisende mit eVisa dürfen sich bis zu 30 Tage im Land aufhalten.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visumgebühren</a></p><div>Die Visumgebühren sind in Euro oder US-Dollar bar zu zahlen.</div>",
						Language:    typesv1.Language_LANGUAGE_DE,
					}},
					LastSignificantUpdate: timestamppb.New(time.Now()),
					Status:                infov1.ItemStatus_ITEM_STATUS_TRUE,
				},
				},
			}},
		}},
		Items: []*infov1.CountryEntryRequirementItem{
			{
				Key: "EntryDocumentsRequired",
				Info: []*infov1.LocalizedItemInfo{
					{
						Name:        "Entry forms",
						Description: "<div><p>Individuals must fill out a <a href=\"https://www.egyptair.com/en/about-egyptair/news-and-press/Documents/%D8%A7%D9%84%D8%A7%D9%95%D9%82%D8%B1%D8%A7%D8%B1%20%D8%A7%D9%84%D8%B5%D8%AD%D9%8A%20%D9%84%D8%BA%D9%8A%D8%B1%20%D8%A7%D9%84%D9%85%D8%B5%D8%B1%D9%8A%D9%8A%D9%86%20%28%D8%A7%D9%84%D8%A7%D9%94%D8%AC%D8%A7%D9%86%D8%A8%29.pdf\" rel=\"noopener noreferrer\" target=\"_blank\">health form</a> upon entry, which they can complete either at the airport, on the plane, or beforehand.</p></div>",
						Language:    typesv1.Language_LANGUAGE_EN,
					},
					{
						Name:        "Einreiseformulare",
						Description: "<div><div><p>Personen müssen bei Einreise ein <a href=\"https://www.egyptair.com/en/about-egyptair/news-and-press/Documents/%D8%A7%D9%84%D8%A7%D9%95%D9%82%D8%B1%D8%A7%D8%B1%20%D8%A7%D9%84%D8%B5%D8%AD%D9%8A%20%D9%84%D8%BA%D9%8A%D8%B1%20%D8%A7%D9%84%D9%85%D8%B5%D8%B1%D9%8A%D9%8A%D9%86%20%28%D8%A7%D9%84%D8%A7%D9%94%D8%AC%D8%A7%D9%86%D8%A8%29.pdf\" rel=\"noopener noreferrer\" target=\"_blank\">Gesundheitsformular</a> abgeben, welches entweder am Flughafen, im Flugzeug oder vor Antritt der Reise ausfüllen.</p></div></div>",
						Language:    typesv1.Language_LANGUAGE_DE,
					},
				},
				LastSignificantUpdate: timestamppb.New(time.Now()),
				Status:                infov1.ItemStatus_ITEM_STATUS_FALSE,
			},
			{
				Key: "ErVisaText",
				Info: []*infov1.LocalizedItemInfo{
					{
						Name:        "Visa required for stay",
						Description: "<div><p>A visa is required for the stay. This can be applied for as an e-Visa or on arrival as a \"Visa on Arrival\". </p></div><div><div>Travellers with eVisa are permitted to stay up to 30 days in Egypt.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visa fees</a></p><div>Visa fees must be paid in cash in euros or US dollars.</div>",
						Language:    typesv1.Language_LANGUAGE_EN,
					},
					{
						Name:        "Visum erforderlich für Aufenthalt",
						Description: "<div><p>Es ist ein Visum für den Aufenthalt erforderlich. Dieses kann als e-Visum oder bei Ankunft als \"Visa on Arrival\" beantragt werden. </p></div><div><div>Reisende mit eVisa dürfen sich bis zu 30 Tage im Land aufhalten.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visumgebühren</a></p><div>Die Visumgebühren sind in Euro oder US-Dollar bar zu zahlen.</div>",
						Language:    typesv1.Language_LANGUAGE_DE,
					},
				},
				LastSignificantUpdate: timestamppb.New(time.Now()),
				Status:                infov1.ItemStatus_ITEM_STATUS_TRUE,
			},
		},
	}
	return &response, nil
}
func (p *partnerPlugin) ActivityProductInfo(ctx context.Context, request *activityv1.ActivityProductInfoRequest) (*activityv1.ActivityProductInfoResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	grpc.SendHeader(ctx, md.ToGrpcMD())
	response := activityv1.ActivityProductInfoResponse{
		Header: nil,
		Activities: []*activityv1.ActivityExtendedInfo{
			{
				Activity: &activityv1.Activity{
					Context:      "DE123456789",
					LastModified: timestamppb.New(time.Now()),
				},
				Units:        []*activityv1.ActivityUnit{},
				Services:     []*activityv1.ActivityService{},
				Zones:        []*activityv1.TransferZone{},
				Descriptions: []*typesv1.LocalizedDescriptionSet{},
				Location:     &activityv1.ActivityLocation{},
				Features:     []*activityv1.ActivityFeature{},
				Tags:         []*activityv1.ActivityTag{},
				Images:       []*typesv1.Image{},
				//TODO: @VjeraTurk add representative example
			},
		},
	}
	return &response, nil
}

func main() {
	grpcServer := grpc.NewServer()
	activityv1grpc.RegisterActivitySearchServiceServer(grpcServer, &partnerPlugin{})
	accommodationv1grpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &partnerPlugin{})
	accommodationv1grpc.RegisterAccommodationProductListServiceServer(grpcServer, &partnerPlugin{})
	accommodationv1grpc.RegisterAccommodationSearchServiceServer(grpcServer, &partnerPlugin{})
	networkv1grpc.RegisterGetNetworkFeeServiceServer(grpcServer, &partnerPlugin{})
	partnerv1grpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, &partnerPlugin{})
	bookv1grpc.RegisterMintServiceServer(grpcServer, &partnerPlugin{})
	bookv1grpc.RegisterValidationServiceServer(grpcServer, &partnerPlugin{})
	pingv1grpc.RegisterPingServiceServer(grpcServer, &partnerPlugin{})
	transportv1grpc.RegisterTransportSearchServiceServer(grpcServer, &partnerPlugin{})
	seat_mapv1grpc.RegisterSeatMapServiceServer(grpcServer, &partnerPlugin{})
	seat_mapv1grpc.RegisterSeatMapAvailabilityServiceServer(grpcServer, &partnerPlugin{})
	infov1grpc.RegisterCountryEntryRequirementsServiceServer(grpcServer, &partnerPlugin{})
	activityv1grpc.RegisterActivityProductInfoServiceServer(grpcServer, &partnerPlugin{})
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
	grpcServer.Serve(lis)
}
