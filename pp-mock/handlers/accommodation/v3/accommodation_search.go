// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"context"
	"fmt"
	"log"
	"math"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v3/accommodationv3grpc"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	"github.com/chain4travel/camino-messenger-bot/v11/pkg/metadata"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/events"
	common "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var _ accommodationv3grpc.AccommodationSearchServiceServer = (*accommodationSearchV3Server)(nil)

type accommodationSearchV3Server struct {
	eventSender events.Sender
}

func NewAccommodationSearchV3Server(eventSender events.Sender) accommodationv3grpc.AccommodationSearchServiceServer {
	return &accommodationSearchV3Server{eventSender: eventSender}
}

func (s *accommodationSearchV3Server) AccommodationSearch(ctx context.Context, req *accommodationv3.AccommodationSearchRequest) (*accommodationv3.AccommodationSearchResponse, error) {
	if err := s.eventSender.SendProtoEvent(req); err != nil {
		log.Printf("error sending event: %v", err)
	}

	md := metadata.Metadata{}

	fmt.Printf("Search generic params: %+v\n", req.SearchParametersGeneric)

	if err := md.ExtractMetadata(ctx); err != nil {
		// TODO @evlekht Improve error handling for metadata extraction - handle consistently across all files. Must either return error or error response.
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))
	log.Printf("Responding to request (Accommodation Search): %s", md.RequestID)

	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &accommodationv3.AccommodationSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "No queries provided",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// check if SearchParametersGeneric is nil or if Currency is nil
	if req.SearchParametersGeneric == nil || req.SearchParametersGeneric.Currency == nil {
		return &accommodationv3.AccommodationSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{{
					Message: "Mandatory field SearchParametersGeneric.Currency is missing",
					Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
				}},
			},
		}, nil
	}

	// loop queries and check if there is travel period
	for _, query := range req.Queries {
		if query.TravelPeriod == nil {
			return &accommodationv3.AccommodationSearchResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{{
						Message: "Mandatory field TravelPeriod is missing. A travel period is required to search for accommodations (with limits of start/end values of now() / now() + 60 days)",
						Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					}},
				},
			}, nil
		}

		if !common.IsTravelPeriodAllowed(query.TravelPeriod) {
			return &accommodationv3.AccommodationSearchResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{{
						Message: "Travel period is outside of the allowed constraints. The range is now() - now()+60 days. Additionally the start date must be before the end date.",
						Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					}},
				},
			}, nil
		}
	}

	// edge-case prevention: check if the traveller definition is identical
	// in all queries. If not return an "unsupported" error.
	unsupportedResp := &accommodationv3.AccommodationSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
			Alerts: []*typesv1.Alert{{
				Message: "Unsupported: Traveller definitions must be identical in all queries",
				Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
			}},
		},
	}
	for queryIndex, query := range req.Queries {
		for queryIndex2, query2 := range req.Queries {
			if queryIndex != queryIndex2 {
				travellersA := query.GetTravellers()
				travellersB := query2.GetTravellers()

				if len(travellersA) != len(travellersB) {
					return unsupportedResp, nil
				}

				for i, travellerA := range travellersA {
					travellerB := travellersB[i]
					if !proto.Equal(travellerA, travellerB) {
						return unsupportedResp, nil
					}
				}
			}
		}
	}

	searchResults := []*accommodationv3.AccommodationSearchResult{}
	resultIDnum := int32(1)
	validationPrices := []*state.UnifiedPrice{}

	// loop request queries
	for _, query := range req.Queries {
		filteredProps := filterExtendedPropertiesByGeoTreeLocation(mockdata.PropertiesV3, query.SearchParametersAccommodation.GetLocationGeoTree())
		filteredProps = filterExtendedPropertiesByProductCodes(filteredProps, query.SearchParametersAccommodation.GetProductCodes())
		filteredProps = filterExtendedPropertiesBySupplierCodes(filteredProps, query.SearchParametersAccommodation.GetSupplierCodes())

		// extract the duration of the travel period in days
		// and round up the result to full days
		duration := common.DateV1ToTime(query.TravelPeriod.GetEndDate()).Sub(common.DateV1ToTime(query.TravelPeriod.GetStartDate())).Hours() / 24
		duration = math.Ceil(duration)

		// generate search result
		for _, prop := range filteredProps {
			units := []*accommodationv3.Unit{}
			// loop all rooms
			for _, room := range prop.Rooms {
				units = append(units, &accommodationv3.Unit{
					Type:             accommodationv3.UnitType(prop.Property.CategoryUnit),
					SupplierRoomCode: room.SupplierCode,
					SupplierRoomName: room.SupplierName,
					OriginalRoomName: room.OriginalName,
					TravelPeriod: &typesv1.TravelPeriod{
						StartDate: &typesv1.Date{
							Year:  query.TravelPeriod.GetStartDate().GetYear(),
							Month: query.TravelPeriod.GetStartDate().GetMonth(),
							Day:   query.TravelPeriod.GetStartDate().GetDay(),
						},
						EndDate: &typesv1.Date{
							Year:  query.TravelPeriod.GetEndDate().GetYear(),
							Month: query.TravelPeriod.GetEndDate().GetMonth(),
							Day:   query.TravelPeriod.GetEndDate().GetDay(),
						},
					},
					TravellerIds: getTravellerIDs(query.Travellers),
					Beds:         room.Beds,
					PriceDetail: &typesv3.PriceDetail{
						Price: &typesv3.Price{
							Value:    fmt.Sprintf("%.0f", common.DefaultPricePerNight*100),
							Decimals: 2,
							Currency: common.CloneProto(req.SearchParametersGeneric.Currency),
						},
						Description: "price per night",
					},
					Services:       []*typesv3.ServiceFact{},
					MealPlanCode:   &typesv1.MealPlan{},
					RatePlan:       &typesv1.RatePlan{},
					RateRule:       &typesv1.RateRule{},
					CancelPolicies: []*typesv3.CancelPolicy{},
					RemainingUnits: 0,
					PropertyCode:   &typesv2.ProductCode{},
					SupplierCode:   prop.Property.SupplierCode,
					Remarks:        "",
				})
			}

			searchPrice := &typesv3.Price{
				Value:    fmt.Sprintf("%.0f", common.DefaultPricePerNight*duration*100),
				Decimals: 2,
				Currency: common.CloneProto(req.SearchParametersGeneric.Currency),
			}
			searchResults = append(searchResults, &accommodationv3.AccommodationSearchResult{
				ResultId: resultIDnum,
				QueryId:  query.QueryId,
				TotalPriceDetail: &typesv3.PriceDetail{
					Price: searchPrice,
				},
				Units: units,
			})

			validationPrice := state.PriceV3ToUnifiedPrice(searchPrice)
			validationPrices = append(validationPrices, validationPrice)

			resultIDnum++
		}
	}

	response := &accommodationv3.AccommodationSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Results: searchResults,
	}

	if len(searchResults) == 0 {
		response.Header.Alerts = []*typesv1.Alert{{
			Message: fmt.Sprintf("No results found for search %v", req.Queries),
			Type:    typesv1.AlertType_ALERT_TYPE_INFO,
		}}
	} else {
		response.Metadata = &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: uuid.New().String()},
		}
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.RecipientCMAccount, md.SenderCMAccount)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
		NumResults:   len(searchResults),
		NumTravelers: len(req.Queries[0].Travellers),
		Prices:       validationPrices,
		JSONRequest:  req.String(),
		JSONResponse: response.String(),
	})

	return response, nil
}

// Extracts traveller IDs from []*typesv3.BasicTraveller
func getTravellerIDs(travellers []*typesv3.BasicTraveller) []int32 {
	ids := make([]int32, len(travellers))
	for i := range travellers {
		ids[i] = travellers[i].TravellerId
	}
	return ids
}
