// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package v2

import (
	"context"
	"fmt"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/common"
	"github.com/chain4travel/camino-messenger-bot/v11/pp-mock/handlers/state"
	mockdata "github.com/chain4travel/camino-messenger-bot/v11/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

var _ accommodationv2grpc.AccommodationSearchServiceServer = (*accommodationSearchV2Server)(nil)

type accommodationSearchV2Server struct{}

func NewAccommodationSearchV2Server() accommodationv2grpc.AccommodationSearchServiceServer {
	return &accommodationSearchV2Server{}
}

func (s *accommodationSearchV2Server) AccommodationSearch(_ context.Context, req *accommodationv2.AccommodationSearchRequest) (*accommodationv2.AccommodationSearchResponse, error) {
	fmt.Printf("Search generic params: %+v\n", req.SearchParametersGeneric)
	// if there is no query, return no results
	if len(req.Queries) == 0 {
		return &accommodationv2.AccommodationSearchResponse{
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
		return &accommodationv2.AccommodationSearchResponse{
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
			return &accommodationv2.AccommodationSearchResponse{
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
			return &accommodationv2.AccommodationSearchResponse{
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
	unsupportedResp := &accommodationv2.AccommodationSearchResponse{
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

	searchResults := []*accommodationv2.AccommodationSearchResult{}
	resultIDnum := int32(1)
	validationPrices := []*state.UnifiedPrice{}

	// loop request queries
	for _, query := range req.Queries {
		filteredProps := filterExtendedPropertiesByGeoTreeLocation(mockdata.PropertiesV2, query.SearchParametersAccommodation.GetLocationGeoTree())
		filteredProps = filterExtendedPropertiesByProductCodes(filteredProps, query.SearchParametersAccommodation.GetProductCodes())
		filteredProps = filterExtendedPropertiesBySupplierCodes(filteredProps, query.SearchParametersAccommodation.GetSupplierCodes())

		// extract the duration of the travel period in days
		duration := common.DaysBetweenDates(query.TravelPeriod.GetEndDate(), query.TravelPeriod.GetStartDate())

		// generate search result
		for _, prop := range filteredProps {
			// empty units array
			units := []*accommodationv2.Unit{}
			// loop all rooms
			for _, room := range prop.Rooms {
				units = append(units, &accommodationv2.Unit{
					Type:             accommodationv2.UnitType(prop.Property.CategoryUnit),
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
					PriceDetail: &typesv2.PriceDetail{
						Price: &typesv2.Price{
							Value:    common.DefaultPricePerNightStr,
							Decimals: common.DefaultPricePerNightDecimals,
							Currency: common.CloneProto(req.SearchParametersGeneric.Currency),
						},
						Description: "price per night",
					},
					Services:       []*typesv2.ServiceFact{},
					MealPlanCode:   &typesv1.MealPlan{},
					RatePlan:       &typesv1.RatePlan{},
					RateRule:       &typesv1.RateRule{},
					CancelPolicies: []*typesv2.CancelPolicy{},
					RemainingUnits: 0,
					PropertyCode:   &typesv2.ProductCode{},
					SupplierCode:   prop.Property.SupplierCode,
					Remarks:        "",
				})
			}

			searchPrice := &typesv2.Price{
				Value:    fmt.Sprintf("%d", common.DefaultPricePerNight*duration),
				Decimals: common.DefaultPricePerNightDecimals,
				Currency: common.CloneProto(req.SearchParametersGeneric.Currency),
			}

			searchResults = append(searchResults, &accommodationv2.AccommodationSearchResult{
				ResultId: resultIDnum,
				QueryId:  query.QueryId,
				TotalPriceDetail: &typesv2.PriceDetail{
					Price: searchPrice,
				},
				Units: units,
			})

			validationPrice := state.PriceV2ToUnifiedPrice(searchPrice)
			validationPrices = append(validationPrices, validationPrice)

			resultIDnum++
		}
	}

	response := &accommodationv2.AccommodationSearchResponse{
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
		response.Metadata = &typesv2.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: uuid.New().String()},
		}
		state.GetStore().AddSearchResult(response.Metadata.SearchId.Value, state.SearchData{
			NumResults:   len(searchResults),
			NumTravelers: len(req.Queries[0].Travellers),
			Prices:       validationPrices,
			JSONRequest:  req.String(),
			JSONResponse: response.String(),
		})
	}

	return response, nil
}

// Extracts traveller IDs from []*typesv2.BasicTraveller
func getTravellerIDs(travellers []*typesv2.BasicTraveller) []int32 {
	ids := make([]int32, len(travellers))
	for i := range travellers {
		ids[i] = travellers[i].TravellerId
	}
	return ids
}
