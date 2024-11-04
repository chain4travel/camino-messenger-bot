package handlers

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	helpers "github.com/chain4travel/camino-messenger-bot/examples/rpc/partner-plugin/helpers"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
)

var _ accommodationv1grpc.AccommodationSearchServiceServer = (*AccommodationSearchV1Server)(nil)

type AccommodationSearchV1Server struct{}

func (*AccommodationSearchV1Server) AccommodationSearch(ctx context.Context, req *accommodationv1.AccommodationSearchRequest) (*accommodationv1.AccommodationSearchResponse, error) {
	md := metadata.Metadata{}

	var params = req.SearchParametersGeneric
	// print params
	fmt.Printf("params: %+v\n", params)

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Search): %s", md.RequestID)

	// load mock data
	properties := helpers.LoadPropertiesMockData()

	// log
	fmt.Printf("properties: %+v\n", properties)

	// Optional: Verify by reading back and printing
	var searchResults []*accommodationv1.AccommodationSearchResult

	// Filter search results by unit supplier room name
	searchResults = filterSearchResultsBySupplierRoomCode(searchResults, "2")

	response := &accommodationv1.AccommodationSearchResponse{
		Header: nil,
		Metadata: &typesv1.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: md.RequestID},
		},
		Results: searchResults,
		Travellers: []*typesv1.BasicTraveller{{
			Type:        typesv1.TravellerType(typesv1.TravelType_TRAVEL_TYPE_LEISURE),
			Birthdate:   &typesv1.Date{},
			Nationality: typesv1.Country_COUNTRY_DE,
		}},
	}

	log.Printf("CMAccount %s received request from CMAccount %s", md.Recipient, md.Sender)

	grpc.SendHeader(ctx, md.ToGrpcMD())

	return response, nil
}

func filterSearchResultsBySupplierRoomCode(results []*accommodationv1.AccommodationSearchResult, pattern string) []*accommodationv1.AccommodationSearchResult {
	regex := regexp.MustCompile(pattern)
	filtered := make([]*accommodationv1.AccommodationSearchResult, 0)
	for _, result := range results {
		for _, unit := range result.Units {
			if regex.MatchString(unit.SupplierRoomCode) {
				filtered = append(filtered, result)
				break
			}
		}
	}
	return filtered
}
