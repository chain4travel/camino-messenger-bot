package handlers

import (
	"context"
	"fmt"
	"log"
	"os"

	"buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/accommodation/v1/accommodationv1grpc"
	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	"github.com/chain4travel/camino-messenger-bot/internal/metadata"
	"google.golang.org/grpc"
)

var _ accommodationv1grpc.AccommodationSearchServiceServer = (*AccommodationSearchV1Server)(nil)

type AccommodationSearchV1Server struct{}

func (*AccommodationSearchV1Server) AccommodationSearch(ctx context.Context, _ *accommodationv1.AccommodationSearchRequest) (*accommodationv1.AccommodationSearchResponse, error) {
	md := metadata.Metadata{}

	if err := md.ExtractMetadata(ctx); err != nil {
		log.Print("error extracting metadata")
	}

	md.Stamp(fmt.Sprintf("%s-%s", "ext-system", "response"))

	log.Printf("Responding to request (Accommodation Search): %s", md.RequestID)

	// Get absolute path based on current file location
	// absPath, err := filepath.Abs("../../examples/rpc/partner-plugin/mock_data/accommodation_search.json")
	// if err != nil {
	// 	log.Fatalf("Error constructing absolute path: %v", err)
	// }

	// file, err := os.ReadFile(absPath)
	// if err != nil {
	// 	log.Fatalf("Failed to read file: %v", err)
	// }

	// Print or save the serialized protobuf data
	// fmt.Println(protoData)

	// Write to a file using os package
	if err := os.WriteFile("output.pb", protoData, 0644); err != nil {
		log.Fatalf("Failed to write protobuf to file: %v", err)
	}

	// Create a new Person message
	// response := &accommodationv1.AccommodationSearchResponse{}

	// // Convert JSON to protobuf
	// err = protojson.Unmarshal(data, response)
	// if err != nil {
	// 	log.Fatalf("Error unmarshaling JSON to protobuf: %v", err)
	// }

	fmt.Println("Successfully converted JSON to protobuf")

	// Optional: Verify by reading back and printing

	response := accommodationv1.AccommodationSearchResponse{
		Header: nil,
		Metadata: &typesv1.SearchResponseMetadata{
			SearchId: &typesv1.UUID{Value: md.RequestID},
		},
		Results: []*accommodationv1.AccommodationSearchResult{{
			ResultId: 0,
			QueryId:  0,
			Units: []*accommodationv1.Unit{{
				Type:             *accommodationv1.UnitType_UNIT_TYPE_ROOM.Enum(),
				SupplierRoomCode: "RMSDDB0000",
				SupplierRoomName: "Double Standard Room",
				OriginalRoomName: "Room with a view",
				TravelPeriod:     &typesv1.TravelPeriod{},
				// TravellerIds:
			}},
			TotalPriceDetail: &typesv1.PriceDetail{
				Price: &typesv1.Price{
					Currency: &typesv1.Currency{
						Currency: &typesv1.Currency_NativeToken{},
					},
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
			CancelPolicy: &typesv1.CancelPolicy{},
			Bookability:  &typesv1.Bookability{},
			Remarks:      "A remark",
		}},
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
