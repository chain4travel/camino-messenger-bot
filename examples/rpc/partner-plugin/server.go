package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1alpha/accommodationv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/activity/v1alpha/activityv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/book/v1alpha/bookv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/info/v1alpha/infov1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/network/v1alpha/networkv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/partner/v1alpha/partnerv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/ping/v1alpha/pingv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/seat_map/v1alpha/seat_mapv1alphagrpc"
	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/transport/v1alpha/transportv1alphagrpc"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	accommodationv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1alpha"
	activityv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1alpha"
	bookv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/book/v1alpha"
	infov1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/info/v1alpha"
	networkv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/network/v1alpha"
	partnerv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/partner/v1alpha"
	pingv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/ping/v1alpha"
	seat_mapv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/seat_map/v1alpha"
	transportv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1alpha"
	typesv1alpha "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1alpha"
)

type partnerPlugin struct {
	activityv1alphagrpc.ActivitySearchServiceServer
	accommodationv1alphagrpc.AccommodationProductInfoServiceServer
	accommodationv1alphagrpc.AccommodationProductListServiceServer
	accommodationv1alphagrpc.AccommodationSearchServiceServer
	networkv1alphagrpc.GetNetworkFeeServiceServer
	partnerv1alphagrpc.GetPartnerConfigurationServiceServer
	pingv1alphagrpc.PingServiceServer
	transportv1alphagrpc.TransportSearchServiceServer
	seat_mapv1alphagrpc.SeatMapServiceServer
	seat_mapv1alphagrpc.SeatMapAvailabilityServiceServer
	infov1alphagrpc.CountryEntryRequirementsServiceServer
}

func (p *partnerPlugin) Mint(ctx context.Context, _ *bookv1alpha.MintRequest) (*bookv1alpha.MintResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := bookv1alpha.MintResponse{
		MintId: &typesv1alpha.UUID{Value: md.RequestID},
		BuyableUntil: &timestamppb.Timestamp{
			Seconds: time.Now().Add(5 * time.Minute).Unix(),
		},
		Price: &typesv1alpha.Price{
			Value:    "1",
			Decimals: 9,
		},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) Validation(ctx context.Context, _ *bookv1alpha.ValidationRequest) (*bookv1alpha.ValidationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := bookv1alpha.ValidationResponse{
		Header:           nil,
		ValidationId:     &typesv1alpha.UUID{Value: md.RequestID},
		ValidationObject: nil,
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) ActivitySearch(ctx context.Context, _ *activityv1alpha.ActivitySearchRequest) (*activityv1alpha.ActivitySearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := activityv1alpha.ActivitySearchResponse{
		Header:   nil,
		Metadata: &typesv1alpha.SearchResponseMetadata{SearchId: &typesv1alpha.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) AccommodationProductInfo(ctx context.Context, _ *accommodationv1alpha.AccommodationProductInfoRequest) (*accommodationv1alpha.AccommodationProductInfoResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := accommodationv1alpha.AccommodationProductInfoResponse{
		Properties: []*accommodationv1alpha.PropertyExtendedInfo{{PaymentType: "cash"}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) AccommodationProductList(ctx context.Context, _ *accommodationv1alpha.AccommodationProductListRequest) (*accommodationv1alpha.AccommodationProductListResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := accommodationv1alpha.AccommodationProductListResponse{
		Properties: []*accommodationv1alpha.Property{{Name: "Hotel"}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) AccommodationSearch(ctx context.Context, _ *accommodationv1alpha.AccommodationSearchRequest) (*accommodationv1alpha.AccommodationSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := accommodationv1alpha.AccommodationSearchResponse{
		Header:   nil,
		Metadata: &typesv1alpha.SearchResponseMetadata{SearchId: &typesv1alpha.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) GetNetworkFee(ctx context.Context, request *networkv1alpha.GetNetworkFeeRequest) (*networkv1alpha.GetNetworkFeeResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := networkv1alpha.GetNetworkFeeResponse{
		NetworkFee: &networkv1alpha.NetworkFee{
			Amount: 0,
		},
		CurrentBlockHeight: request.BlockHeight,
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) GetPartnerConfiguration(ctx context.Context, request *partnerv1alpha.GetPartnerConfigurationRequest) (*partnerv1alpha.GetPartnerConfigurationResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := partnerv1alpha.GetPartnerConfigurationResponse{
		PartnerConfiguration: &partnerv1alpha.PartnerConfiguration{},
		CurrentBlockHeight:   request.GetBlockHeight(),
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) Ping(ctx context.Context, request *pingv1alpha.PingRequest) (*pingv1alpha.PingResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	return &pingv1alpha.PingResponse{
		Header:      nil,
		PingMessage: fmt.Sprintf("Ping response to [%s] with request ID: %s", request.PingMessage, md.RequestID),
	}, nil
}

func (p *partnerPlugin) TransportSearch(ctx context.Context, _ *transportv1alpha.TransportSearchRequest) (*transportv1alpha.TransportSearchResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := transportv1alpha.TransportSearchResponse{
		Header:   nil,
		Metadata: &typesv1alpha.SearchResponseMetadata{SearchId: &typesv1alpha.UUID{Value: md.RequestID}},
	}
	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}
func (p *partnerPlugin) SeatMap(ctx context.Context, request *seat_mapv1alpha.SeatMapRequest) (*seat_mapv1alpha.SeatMapResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := seat_mapv1alpha.SeatMapResponse{
		Header: nil,
		SeatMap: &typesv1alpha.SeatMap{
			Id: md.RequestID,
			Sections: []*typesv1alpha.Section{{
				Id: "123ST",
				Names: []*typesv1alpha.LocalizedString{{
					Language: typesv1alpha.Language_LANGUAGE_EN,
					Text:     "North Stand",
				},
					{
						Language: typesv1alpha.Language_LANGUAGE_DE,
						Text:     "Nordtribüne",
					},
				},
				SeatInfo: &typesv1alpha.Section_SeatList{
					SeatList: &typesv1alpha.SeatList{
						Seats: []*typesv1alpha.Seat{
							{
								Id: "1A",
								Location: &typesv1alpha.SeatLocation{
									Location: &typesv1alpha.SeatLocation_Vector{
										Vector: &typesv1alpha.VectorSeatLocation{
											Label: "section-North-Stand-26-34-2-label",
										},
									},
								},
							},
							{
								Id: "2A",
								Location: &typesv1alpha.SeatLocation{
									Location: &typesv1alpha.SeatLocation_Vector{
										Vector: &typesv1alpha.VectorSeatLocation{
											Label: "section-North-Stand-26-34-2-label",
										},
									},
								},
								Restrictions: []*typesv1alpha.LocalizedSeatAttributeSet{
									{
										Language: typesv1alpha.Language_LANGUAGE_EN,
										SeatAttributes: []*typesv1alpha.SeatAttribute{
											{
												Name:        "Restricted Vision",
												Description: "Seat behind a column",
											},
										},
									},
								},
								Features: []*typesv1alpha.LocalizedSeatAttributeSet{
									{
										Language: typesv1alpha.Language_LANGUAGE_EN,
										SeatAttributes: []*typesv1alpha.SeatAttribute{
											{
												Name:        "Discount",
												Description: "Discount due to restricted vision up to 80%",
												Value:       int32(80),
											},
										},
									},
									{
										Language: typesv1alpha.Language_LANGUAGE_DE,
										SeatAttributes: []*typesv1alpha.SeatAttribute{
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
				Image: &typesv1alpha.Image{
					File: &typesv1alpha.File{
						Name:         "String",
						Url:          "https://camino.network/static/images/6HibYS9gzR-1800.webp", //TODO: replace with an actual image
						LastModified: timestamppb.New(time.Now()),
					},
					Width:  50,
					Height: 50,
				},
				LocalizedDescriptions: []*typesv1alpha.LocalizedDescriptionSet{{
					Language: typesv1alpha.Language_LANGUAGE_EN,
					Descriptions: []*typesv1alpha.Description{{
						Category: "General",
						Text:     "Leather Seats",
					}},
				},
				},
			},
				{
					Id: "124ST",
					Names: []*typesv1alpha.LocalizedString{{
						Language: typesv1alpha.Language_LANGUAGE_EN,
						Text:     "East Stand",
					}, {
						Language: typesv1alpha.Language_LANGUAGE_DE,
						Text:     "Osttribüne",
					}},
					SeatInfo: &typesv1alpha.Section_SeatList{
						SeatList: &typesv1alpha.SeatList{
							Seats: []*typesv1alpha.Seat{
								{
									Id: "31F",
									Location: &typesv1alpha.SeatLocation{
										Location: &typesv1alpha.SeatLocation_Vector{
											Vector: &typesv1alpha.VectorSeatLocation{
												Label: "section-East-Stand-26-34-2-label",
											},
										},
									},
								},
								{
									Id: "32F",
									Location: &typesv1alpha.SeatLocation{
										Location: &typesv1alpha.SeatLocation_Vector{
											Vector: &typesv1alpha.VectorSeatLocation{
												Label: "section-East-Stand-26-34-2-label",
											},
										},
									},
								},
							},
						},
					},
					Image: &typesv1alpha.Image{
						File: &typesv1alpha.File{
							Name:         "String",
							Url:          "https://camino.network/static/images/6HibYS9gzR-1800.webp",
							LastModified: timestamppb.New(time.Now()),
						},
						Width:  50,
						Height: 50,
					},
					LocalizedDescriptions: []*typesv1alpha.LocalizedDescriptionSet{{
						Language: typesv1alpha.Language_LANGUAGE_EN,
						Descriptions: []*typesv1alpha.Description{{
							Category: "General",
							Text:     "Seats",
						}},
					}, {
						Language: typesv1alpha.Language_LANGUAGE_DE,
						Descriptions: []*typesv1alpha.Description{{
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
func (p *partnerPlugin) SeatMapAvailability(ctx context.Context, request *seat_mapv1alpha.SeatMapAvailabilityRequest) (*seat_mapv1alpha.SeatMapAvailabilityResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	response := seat_mapv1alpha.SeatMapAvailabilityResponse{
		Header: nil,
		SeatMap: &typesv1alpha.SeatMapInventory{
			Id: "123ST",
			Sections: []*typesv1alpha.SectionInventory{
				{
					Id: "A",
					SeatInfo: &typesv1alpha.SectionInventory_SeatList{
						SeatList: &typesv1alpha.SeatInventory{
							Ids: []string{"1A", "1B"},
						},
					}},
				{
					Id:       "B",
					SeatInfo: &typesv1alpha.SectionInventory_SeatCount{SeatCount: &wrapperspb.Int32Value{Value: 32}},
				}},
		},
	}

	grpc.SendHeader(ctx, md.ToGrpcMD())
	return &response, nil
}

func (p *partnerPlugin) CountryEntryRequirements(ctx context.Context, request *infov1alpha.CountryEntryRequirementsRequest) (*infov1alpha.CountryEntryRequirementsResponse, error) {
	md := metadata.Metadata{}
	err := md.ExtractMetadata(ctx)
	if err != nil {
		log.Print("error extracting metadata")
	}
	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request: %s", md.RequestID)

	grpc.SendHeader(ctx, md.ToGrpcMD())
	response := infov1alpha.CountryEntryRequirementsResponse{
		Header: nil,
		Categories: []*infov1alpha.CountryEntryRequirementCategory{{
			Key: "entry",
			Names: []*typesv1alpha.LocalizedString{{
				Text:     "Entry",
				Language: typesv1alpha.Language_LANGUAGE_EN,
			}, {
				Text:     "Einreise",
				Language: typesv1alpha.Language_LANGUAGE_DE,
			}},
			SubCategories: []*infov1alpha.CountryEntryRequirementCategory{{
				Key: "entry_documents",
				Names: []*typesv1alpha.LocalizedString{{
					Text:     "Required entry forms and documents",
					Language: typesv1alpha.Language_LANGUAGE_EN,
				}, {
					Text:     "Erforderliche Formulare und Dokumente für die Einreise",
					Language: typesv1alpha.Language_LANGUAGE_DE,
				}},
				Items: []*infov1alpha.CountryEntryRequirementItem{{
					Key: "ErVisaText",
					Info: []*infov1alpha.LocalizedItemInfo{{
						Name:        "Visa required for stay",
						Description: "<div><p>A visa is required for the stay. This can be applied for as an e-Visa or on arrival as a \"Visa on Arrival\". </p></div><div><div>Travellers with eVisa are permitted to stay up to 30 days in Egypt.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visa fees</a></p><div>Visa fees must be paid in cash in euros or US dollars.</div>",
						Language:    typesv1alpha.Language_LANGUAGE_EN,
					}, {
						Name:        "Visum erforderlich für Aufenthalt",
						Description: "<div><p>Es ist ein Visum für den Aufenthalt erforderlich. Dieses kann als e-Visum oder bei Ankunft als \"Visa on Arrival\" beantragt werden. </p></div><div><div>Reisende mit eVisa dürfen sich bis zu 30 Tage im Land aufhalten.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visumgebühren</a></p><div>Die Visumgebühren sind in Euro oder US-Dollar bar zu zahlen.</div>",
						Language:    typesv1alpha.Language_LANGUAGE_DE,
					}},
					LastSignificantUpdate: timestamppb.New(time.Now()),
					Status:                infov1alpha.ItemStatus_ITEM_STATUS_TRUE,
				},
				},
			}},
		}},
		Items: []*infov1alpha.CountryEntryRequirementItem{
			{
				Key: "EntryDocumentsRequired",
				Info: []*infov1alpha.LocalizedItemInfo{
					{
						Name:        "Entry forms",
						Description: "<div><p>Individuals must fill out a <a href=\"https://www.egyptair.com/en/about-egyptair/news-and-press/Documents/%D8%A7%D9%84%D8%A7%D9%95%D9%82%D8%B1%D8%A7%D8%B1%20%D8%A7%D9%84%D8%B5%D8%AD%D9%8A%20%D9%84%D8%BA%D9%8A%D8%B1%20%D8%A7%D9%84%D9%85%D8%B5%D8%B1%D9%8A%D9%8A%D9%86%20%28%D8%A7%D9%84%D8%A7%D9%94%D8%AC%D8%A7%D9%86%D8%A8%29.pdf\" rel=\"noopener noreferrer\" target=\"_blank\">health form</a> upon entry, which they can complete either at the airport, on the plane, or beforehand.</p></div>",
						Language:    typesv1alpha.Language_LANGUAGE_EN,
					},
					{
						Name:        "Einreiseformulare",
						Description: "<div><div><p>Personen müssen bei Einreise ein <a href=\"https://www.egyptair.com/en/about-egyptair/news-and-press/Documents/%D8%A7%D9%84%D8%A7%D9%95%D9%82%D8%B1%D8%A7%D8%B1%20%D8%A7%D9%84%D8%B5%D8%AD%D9%8A%20%D9%84%D8%BA%D9%8A%D8%B1%20%D8%A7%D9%84%D9%85%D8%B5%D8%B1%D9%8A%D9%8A%D9%86%20%28%D8%A7%D9%84%D8%A7%D9%94%D8%AC%D8%A7%D9%86%D8%A8%29.pdf\" rel=\"noopener noreferrer\" target=\"_blank\">Gesundheitsformular</a> abgeben, welches entweder am Flughafen, im Flugzeug oder vor Antritt der Reise ausfüllen.</p></div></div>",
						Language:    typesv1alpha.Language_LANGUAGE_DE,
					},
				},
				LastSignificantUpdate: timestamppb.New(time.Now()),
				Status:                infov1alpha.ItemStatus_ITEM_STATUS_FALSE,
			},
			{
				Key: "ErVisaText",
				Info: []*infov1alpha.LocalizedItemInfo{
					{
						Name:        "Visa required for stay",
						Description: "<div><p>A visa is required for the stay. This can be applied for as an e-Visa or on arrival as a \"Visa on Arrival\". </p></div><div><div>Travellers with eVisa are permitted to stay up to 30 days in Egypt.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visa fees</a></p><div>Visa fees must be paid in cash in euros or US dollars.</div>",
						Language:    typesv1alpha.Language_LANGUAGE_EN,
					},
					{
						Name:        "Visum erforderlich für Aufenthalt",
						Description: "<div><p>Es ist ein Visum für den Aufenthalt erforderlich. Dieses kann als e-Visum oder bei Ankunft als \"Visa on Arrival\" beantragt werden. </p></div><div><div>Reisende mit eVisa dürfen sich bis zu 30 Tage im Land aufhalten.</div></div><p><a href=\"https://visa2egypt.gov.eg/eVisa/Home\" target=\"_blank\"><div>Electronic Visa Portal</div></a></p><p><a href=\"https://visa2egypt.gov.eg/eVisa/FAQ?VISTK=4N4T-00SQ-1JY3-6SA4-BSGM-RHA8-VTWB-JK1L-PU27-3H7K-Y7CV-C7BX-BH94-A1RD-DW7O-CHD8\" target=\"_blank\">Visumgebühren</a></p><div>Die Visumgebühren sind in Euro oder US-Dollar bar zu zahlen.</div>",
						Language:    typesv1alpha.Language_LANGUAGE_DE,
					},
				},
				LastSignificantUpdate: timestamppb.New(time.Now()),
				Status:                infov1alpha.ItemStatus_ITEM_STATUS_TRUE,
			},
		},
	}
	return &response, nil
}
func main() {
	grpcServer := grpc.NewServer()
	activityv1alphagrpc.RegisterActivitySearchServiceServer(grpcServer, &partnerPlugin{})
	accommodationv1alphagrpc.RegisterAccommodationProductInfoServiceServer(grpcServer, &partnerPlugin{})
	accommodationv1alphagrpc.RegisterAccommodationProductListServiceServer(grpcServer, &partnerPlugin{})
	accommodationv1alphagrpc.RegisterAccommodationSearchServiceServer(grpcServer, &partnerPlugin{})
	networkv1alphagrpc.RegisterGetNetworkFeeServiceServer(grpcServer, &partnerPlugin{})
	partnerv1alphagrpc.RegisterGetPartnerConfigurationServiceServer(grpcServer, &partnerPlugin{})
	bookv1alphagrpc.RegisterMintServiceServer(grpcServer, &partnerPlugin{})
	bookv1alphagrpc.RegisterValidationServiceServer(grpcServer, &partnerPlugin{})
	pingv1alphagrpc.RegisterPingServiceServer(grpcServer, &partnerPlugin{})
	transportv1alphagrpc.RegisterTransportSearchServiceServer(grpcServer, &partnerPlugin{})
	seat_mapv1alphagrpc.RegisterSeatMapServiceServer(grpcServer, &partnerPlugin{})
	seat_mapv1alphagrpc.RegisterSeatMapAvailabilityServiceServer(grpcServer, &partnerPlugin{})
	infov1alphagrpc.RegisterCountryEntryRequirementsServiceServer(grpcServer, &partnerPlugin{})
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
