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
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	common "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var _ accommodationv3grpc.AccommodationSearchServiceServer = (*AccommodationSearchV3Server)(nil)

type AccommodationSearchV3Server struct{}

func (*AccommodationSearchV3Server) AccommodationSearch(ctx context.Context, req *accommodationv3.AccommodationSearchRequest) (*accommodationv3.AccommodationSearchResponse, error) {
	md := metadata.Metadata{}

	searchGenericParams := req.SearchParametersGeneric
	// print params
	fmt.Printf("Search generic params: %+v\n", searchGenericParams)

	if err := md.ExtractMetadata(ctx); err != nil {
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

	// loop queries and check if there is travel period
	for _, query := range req.Queries {
		if query.TravelPeriod == nil {
			return &accommodationv3.AccommodationSearchResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{{
						Message: "Mandatory field TravelPeriod is missing. A travel period is required to search for accommodations (with limits of start/end values of now() / now() + 60 days)",
						Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					},
					}},
			}, nil
		}

		if !common.IsTravelPeriodAllowed(query.TravelPeriod) {
			return &accommodationv3.AccommodationSearchResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{{
						Message: "Travel period is outside of the allowed constraints. The range is now() - now()+60 days. Additionally the start date must be before the end date.",
						Type:    typesv1.AlertType_ALERT_TYPE_ERROR,
					},
					}},
			}, nil
		}
	}

	searchResults := []*accommodationv3.AccommodationSearchResult{}
	resultIDnum := int32(1)

	// loop request queries
	for _, query := range req.Queries {
		availableProperties := []*accommodationv3.PropertyExtendedInfo{}
		// get filtered properties
		filteredProps := filterPropertiesByGeoTreeLocation(mockdata.PropertiesV3, query.SearchParametersAccommodation.GetLocationGeoTree())
		// filter by product codes
		filteredProps = filterPropertiesByProductCodes(filteredProps, query.SearchParametersAccommodation.GetProductCodes())
		// filter by supplier codes
		filteredProps = filterPropertiesBySupplierCodes(filteredProps, query.SearchParametersAccommodation.GetSupplierCodes())

		// loop filtered properties and check if they are already in availableProperties
	filteredPropsLoop:
		for _, prop := range filteredProps {
			// Check if property already exists in availableProperties
			for _, existingProp := range availableProperties {
				if existingProp.Property.SupplierCode.SupplierCode == prop.Property.SupplierCode.SupplierCode {
					continue filteredPropsLoop
				}
			}
			availableProperties = append(availableProperties, prop)
		}

		// extract the duration of the travel period in days
		// and round up the result to full days
		duration := common.DateV1ToTime(query.TravelPeriod.GetEndDate()).Sub(common.DateV1ToTime(query.TravelPeriod.GetStartDate())).Hours() / 24
		duration = math.Ceil(duration)

		// generate search result
		for _, prop := range availableProperties {
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
							Currency: &typesv3.Currency{Currency: &typesv3.Currency_NativeToken{}},
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

			searchResults = append(searchResults, &accommodationv3.AccommodationSearchResult{
				ResultId: resultIDnum,
				QueryId:  query.QueryId,
				TotalPriceDetail: &typesv3.PriceDetail{
					Price: &typesv3.Price{
						Value:    fmt.Sprintf("%.0f", common.DefaultPricePerNight*duration*100),
						Decimals: 2,
					}},
				Units: units,
			})

			resultIDnum++
		}
	}

	if len(searchResults) == 0 {
		return &accommodationv3.AccommodationSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
				Alerts: []*typesv1.Alert{{
					Message: fmt.Sprintf("No results found for search %v", req.Queries),
					Type:    typesv1.AlertType_ALERT_TYPE_INFO,
				}},
			},
		}, nil
	}

	searchID := uuid.New().String()

	response := &accommodationv3.AccommodationSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv3.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: searchID},
		},
		Results: searchResults,
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	if err := grpc.SetHeader(ctx, md.ToGrpcMD()); err != nil {
		log.Printf("Failed to set header: %v", err)
	}

	return response, nil
}

// FilterPropertiesByGeoTreeLocation filters properties based on city or resort
func filterPropertiesByGeoTreeLocation(
	properties []*accommodationv3.PropertyExtendedInfo,
	geoTreeLocation *typesv2.GeoTree,
) []*accommodationv3.PropertyExtendedInfo {
	if geoTreeLocation == nil ||
		geoTreeLocation.CityOrResort == "" ||
		geoTreeLocation.Region == "" ||
		geoTreeLocation.Country == typesv2.Country_COUNTRY_UNSPECIFIED {
		return properties
	}

	filtered := []*accommodationv3.PropertyExtendedInfo{}
	for _, prop := range properties {
		address := prop.Property.ContactInfo.Address[0]
		if address.GeoTree != nil &&
			address.GeoTree.CityOrResort == geoTreeLocation.CityOrResort &&
			address.GeoTree.Country == geoTreeLocation.Country &&
			address.GeoTree.Region == geoTreeLocation.Region {
			filtered = append(filtered, prop)
		}
	}

	return filtered
}

// Extracts traveller IDs from []*typesv3.BasicTraveller
func getTravellerIDs(travellers []*typesv3.BasicTraveller) []int32 {
	ids := make([]int32, len(travellers))
	for i := range travellers {
		ids[i] = travellers[i].TravellerId
	}
	return ids
}

// filterPropertiesByProductCodes filters properties based on product codes
func filterPropertiesByProductCodes(
	properties []*accommodationv3.PropertyExtendedInfo,
	productCodes []*typesv2.ProductCode,
) []*accommodationv3.PropertyExtendedInfo {
	if len(productCodes) == 0 {
		return properties
	}

	filtered := []*accommodationv3.PropertyExtendedInfo{}
	for _, prop := range properties {
	productCodesLoop:
		for _, code := range productCodes {
			for _, propCode := range prop.Property.ProductCodes {
				if proto.Equal(propCode, code) {
					filtered = append(filtered, prop)
					break productCodesLoop
				}
			}
		}
	}
	return filtered
}

// filterPropertiesBySupplierCodes filters properties based on supplier codes
func filterPropertiesBySupplierCodes(
	properties []*accommodationv3.PropertyExtendedInfo,
	supplierCodes []*typesv2.SupplierProductCode,
) []*accommodationv3.PropertyExtendedInfo {
	if len(supplierCodes) == 0 {
		return properties
	}

	filtered := []*accommodationv3.PropertyExtendedInfo{}
	for _, prop := range properties {
		for _, code := range supplierCodes {
			if prop.Property.SupplierCode.SupplierCode == code.SupplierCode {
				filtered = append(filtered, prop)
				break
			}
		}
	}
	return filtered
}
