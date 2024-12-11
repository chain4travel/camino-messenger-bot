package handlers

import (
	"context"
	"fmt"
	"log"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v2/accommodationv2grpc"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	common "github.com/chain4travel/camino-messenger-bot/pp-mock/handlers"
	mockdata "github.com/chain4travel/camino-messenger-bot/pp-mock/services/data"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

var _ accommodationv2grpc.AccommodationSearchServiceServer = (*AccommodationSearchV2Server)(nil)

type AccommodationSearchV2Server struct{}

func (*AccommodationSearchV2Server) AccommodationSearch(ctx context.Context, req *accommodationv2.AccommodationSearchRequest) (*accommodationv2.AccommodationSearchResponse, error) {
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
		return &accommodationv2.AccommodationSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
				Alerts: []*typesv1.Alert{
					{
						Message: "No queries provided",
						Type:    typesv1.AlertType_ALERT_TYPE_INFO,
					},
				},
			},
		}, nil
	}

	// loop queries and check if there is travel period
	for _, query := range req.Queries {
		if !common.IsTravelPeriodAllowed(query.TravelPeriod) {
			return &accommodationv2.AccommodationSearchResponse{
				Header: &typesv1.ResponseHeader{
					Status: typesv1.StatusType_STATUS_TYPE_FAILURE,
					Alerts: []*typesv1.Alert{
						{
							Message: "No results available for the period",
							Type:    typesv1.AlertType_ALERT_TYPE_INFO,
						},
					},
				},
			}, nil
		}
	}

	searchResults := []*accommodationv2.AccommodationSearchResult{}
	availableProperties := []*accommodationv2.PropertyExtendedInfo{}
	var resultIDnum int32 = 1

	// loop request queries
	for _, query := range req.Queries {
		// get filtered properties
		filteredProps := filterPropertiesByGeoTreeLocation(mockdata.PropertiesV2, query.SearchParametersAccommodation.GetLocationGeoTree())
		// filter by product codes
		filteredProps = filterPropertiesByProductCodes(filteredProps, query.SearchParametersAccommodation.GetProductCodes())
		// filter by supplier codes
		filteredProps = filterPropertiesBySupplierCodes(filteredProps, query.SearchParametersAccommodation.GetSupplierCodes())

		// loop filtered properties and check if they are already in availableProperties
		for _, prop := range filteredProps {
			// Check if property already exists in availableProperties
			exists := false
			for _, existingProp := range availableProperties {
				if existingProp.Property.SupplierCode.SupplierCode == prop.Property.SupplierCode.SupplierCode {
					exists = true
					break
				}
			}
			if !exists {
				availableProperties = append(availableProperties, prop)
			}
		}

		// generate search result
		for _, prop := range availableProperties {
			// empty units array
			units := make([]*accommodationv2.Unit, 0)
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
							Value: common.DefaultPrice,
							Currency: &typesv2.Currency{
								Currency: &typesv2.Currency_NativeToken{},
							},
						},
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

			searchResults = append(searchResults, &accommodationv2.AccommodationSearchResult{
				ResultId: resultIDnum,
				QueryId:  query.QueryId,
				TotalPriceDetail: &typesv2.PriceDetail{
					Price: &typesv2.Price{
						Value: common.DefaultPrice,
					},
				},
				Units: units,
			})

			resultIDnum++
		}
	}

	if len(searchResults) == 0 {
		return &accommodationv2.AccommodationSearchResponse{
			Header: &typesv1.ResponseHeader{
				Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
				Alerts: []*typesv1.Alert{
					{
						Message: fmt.Sprintf("No results found for search %v", req.Queries),
						Type:    typesv1.AlertType_ALERT_TYPE_INFO,
					},
				},
			},
		}, nil
	}

	searchID := uuid.New().String()

	response := &accommodationv2.AccommodationSearchResponse{
		Header: &typesv1.ResponseHeader{
			Status: typesv1.StatusType_STATUS_TYPE_SUCCESS,
		},
		Metadata: &typesv2.SearchResponseMetadata{
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
func filterPropertiesByGeoTreeLocation(properties []*accommodationv2.PropertyExtendedInfo, geoTreeLocation *typesv2.GeoTree) []*accommodationv2.PropertyExtendedInfo {
	if geoTreeLocation == nil || geoTreeLocation.CityOrResort == "" || geoTreeLocation.Region == "" {
		return properties
	}

	filtered := make([]*accommodationv2.PropertyExtendedInfo, 0)
	for _, prop := range properties {
		address := prop.Property.ContactInfo.Address[0]
		if address.GeoTree.CityOrResort == geoTreeLocation.CityOrResort && address.GeoTree.Country == geoTreeLocation.Country && address.GeoTree.Region == geoTreeLocation.Region {
			filtered = append(filtered, prop)
		}
	}

	return filtered
}

// getTravellerIDs extracts traveller IDs from []*typesv2.BasicTraveller
func getTravellerIDs(travellers []*typesv2.BasicTraveller) []int32 {
	// Preallocate slice with exact capacity needed
	ids := make([]int32, 0, len(travellers))
	for _, traveller := range travellers {
		ids = append(ids, traveller.TravellerId)
	}
	return ids
}

// filterPropertiesByProductCodes filters properties based on product codes
func filterPropertiesByProductCodes(properties []*accommodationv2.PropertyExtendedInfo, productCodes []*typesv2.ProductCode) []*accommodationv2.PropertyExtendedInfo {
	if len(productCodes) == 0 {
		return properties
	}

	filtered := make([]*accommodationv2.PropertyExtendedInfo, 0)
	for _, prop := range properties {
		for _, code := range productCodes {
			if prop.Property.ProductCodes[0].Code == code.Code {
				filtered = append(filtered, prop)
				break
			}
		}
	}
	return filtered
}

// filterPropertiesBySupplierCodes filters properties based on supplier codes
func filterPropertiesBySupplierCodes(properties []*accommodationv2.PropertyExtendedInfo, supplierCodes []*typesv2.SupplierProductCode) []*accommodationv2.PropertyExtendedInfo {
	if len(supplierCodes) == 0 {
		return properties
	}

	filtered := make([]*accommodationv2.PropertyExtendedInfo, 0)
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
