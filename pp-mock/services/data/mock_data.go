package mockdata

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	activityv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v1"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
	transportv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v2"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v1"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

//go:embed properties.json
var propertiesJSON []byte

//go:embed tripsv1.json
var tripsV1JSON []byte

//go:embed tripsv3_basic.json
var tripsV3BasicJSON []byte

//go:embed tripsv3_extended.json
var tripsV3ExtendedJSON []byte

//go:embed activityv1.json
var activityV1JSON []byte

//go:embed activityv1_extended.json
var activityExtendedV1JSON []byte

//go:embed activityv1_search.json
var activitySearchResultV1JSON []byte

//go:embed activityv2.json
var activityV2JSON []byte

//go:embed activityv2_extended.json
var activityExtendedV2JSON []byte

//go:embed activityv2_search.json
var activitySearchResultV2JSON []byte

//go:embed activityv3.json
var activityV3JSON []byte

//go:embed activityv3_extended.json
var activityV3ExtendedJSON []byte

//go:embed activityv3_search.json
var activitySearchResultV3JSON []byte

//go:embed seatmapv4/seatmapv4.json
var seatMapV4JSON []byte

//go:embed seatmapv4/seatmap_availability_v4.json
var seatMapAvailabilityV4JSON []byte

const (
	SeatMapTransportIndex = 0
	SeatMapActivityIndex  = 1
)

var (
	PropertiesV1 []*accommodationv1.PropertyExtendedInfo // used by product list, info and search
	PropertiesV2 []*accommodationv2.PropertyExtendedInfo // used by product list, info and search
	PropertiesV3 []*accommodationv3.PropertyExtendedInfo // used by product list, info and search

	TripsV1 []*transportv1.Trip // used by search
	TripsV2 []*transportv2.Trip // used by search

	TripsBasicV3    []*transportv3.TripBasic    // used by product list
	TripsExtendedV3 []*transportv3.TripExtended // used by search

	ActivityV1             []*activityv1.Activity             // used by product list
	ActivityExtendedV1     []*activityv1.ActivityExtendedInfo // used by product info
	ActivitySearchResultV1 []*activityv1.ActivitySearchResult // used by search

	ActivityV2             []*activityv2.Activity             // used by product list
	ActivityExtendedV2     []*activityv2.ActivityExtendedInfo // used by product info
	ActivitySearchResultV2 []*activityv2.ActivitySearchResult // used by search

	ActivityV3             []*activityv3.Activity             // used by product list
	ActivityExtendedV3     []*activityv3.ActivityExtendedInfo // used by product info
	ActivitySearchResultV3 []*activityv3.ActivitySearchResult // used by search

	SeatMapV4             []*typesv4.SeatMap          // used by seatMap
	SeatMapAvailabilityV4 []*typesv4.SeatMapInventory // used by seatMapAvailability

)

func init() {
	var err error

	// because protobuf location and price are one-of interface types,
	// json unmarshaling won't work for them and will result in error
	// so, as quick workaround, we are setting them manually

	if err := json.Unmarshal(propertiesJSON, &PropertiesV1); err != nil {
		panic(fmt.Errorf("error unmarshaling properties v1: %w", err))
	}
	if err := json.Unmarshal(propertiesJSON, &PropertiesV2); err != nil {
		panic(fmt.Errorf("error unmarshaling properties v2: %w", err))
	}
	if err := json.Unmarshal(propertiesJSON, &PropertiesV3); err != nil {
		panic(fmt.Errorf("error unmarshaling properties v3: %w", err))
	}
	if err := json.Unmarshal(tripsV1JSON, &TripsV1); err != nil {
		panic(fmt.Errorf("error unmarshaling trips v1: %w", err))
	}
	if err := json.Unmarshal(tripsV1JSON, &TripsV2); err != nil {
		panic(fmt.Errorf("error unmarshaling trips v2: %w", err))
	}
	if err := json.Unmarshal(tripsV3BasicJSON, &TripsBasicV3); err != nil {
		panic(fmt.Errorf("error unmarshaling trips v3 basic: %w", err))
	}
	if err := json.Unmarshal(tripsV3ExtendedJSON, &TripsExtendedV3); err != nil {
		panic(fmt.Errorf("error unmarshaling trips v3 extended: %w", err))
	}
	if err := json.Unmarshal(activityV1JSON, &ActivityV1); err != nil {
		panic(fmt.Errorf("error unmarshaling activities v1: %w", err))
	}
	if err := json.Unmarshal(activityExtendedV1JSON, &ActivityExtendedV1); err != nil {
		panic(fmt.Errorf("error unmarshaling activities extended v1: %w", err))
	}
	if err := json.Unmarshal(activitySearchResultV1JSON, &ActivitySearchResultV1); err != nil {
		panic(fmt.Errorf("error unmarshaling activities search v1: %w", err))
	}
	if err := json.Unmarshal(activityV2JSON, &ActivityV2); err != nil {
		panic(fmt.Errorf("error unmarshaling activities v2: %w", err))
	}
	if err := json.Unmarshal(activityExtendedV2JSON, &ActivityExtendedV2); err != nil {
		panic(fmt.Errorf("error unmarshaling activities extended v2: %w", err))
	}
	if err := json.Unmarshal(activitySearchResultV2JSON, &ActivitySearchResultV2); err != nil {
		panic(fmt.Errorf("error unmarshaling activities search v2: %w", err))
	}
	if err := json.Unmarshal(activityV3JSON, &ActivityV3); err != nil {
		panic(fmt.Errorf("error unmarshaling activities v3: %w", err))
	}
	if err := json.Unmarshal(activityV3ExtendedJSON, &ActivityExtendedV3); err != nil {
		panic(fmt.Errorf("error unmarshaling activities extended v3: %w", err))
	}
	if err := json.Unmarshal(activitySearchResultV3JSON, &ActivitySearchResultV3); err != nil {
		panic(fmt.Errorf("error unmarshaling activities search v3: %w", err))
	}
	SeatMapV4, err = unmarshalStrictAndValidate[*typesv4.SeatMap](seatMapV4JSON)
	if err != nil {
		panic(fmt.Errorf("error unmarshaling seat map v4: %w", err))
	}
	SeatMapAvailabilityV4, err = unmarshalStrictAndValidate[*typesv4.SeatMapInventory](seatMapAvailabilityV4JSON)
	if err != nil {
		panic(fmt.Errorf("error unmarshaling seat map availability v4: %w", err))
	}

	// TripBasicV3[0,0]
	TripsBasicV3[0].Segments[0].Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "PMI",
				Type: 2,
			},
		},
	}
	TripsBasicV3[0].Segments[0].Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "BCN",
				Type: 2,
			},
		},
	}

	// TripBasicV3[1,0]
	TripsBasicV3[1].Segments[0].Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "BCN",
				Type: 2,
			},
		},
	}
	TripsBasicV3[1].Segments[0].Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "MAD",
				Type: 2,
			},
		},
	}

	// TripBasicV3[1,1]
	TripsBasicV3[1].Segments[1].Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "MAD",
				Type: 2,
			},
		},
	}
	TripsBasicV3[1].Segments[1].Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "LIS",
				Type: 2,
			},
		},
	}

	// TripBasicV3[2,0]
	TripsBasicV3[2].Segments[0].Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "LAN",
				Type: 4,
			},
		},
	}
	TripsBasicV3[2].Segments[0].Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "HAM",
				Type: 4,
			},
		},
	}
	// TripBasicV3[2,1]
	TripsBasicV3[2].Segments[1].Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "HAM",
				Type: 4,
			},
		},
	}
	TripsBasicV3[2].Segments[1].Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "BER",
				Type: 4,
			},
		},
	}

	// TripsExtendedV3[0]
	TripsExtendedV3[0].Price.Currency = &typesv3.Currency{
		Currency: &typesv3.Currency_IsoCurrency{
			IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}

	// TripsExtendedV3[0,0]
	TripsExtendedV3[0].Segments[0].Info.Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "PMI",
				Type: 2,
			},
		},
	}
	TripsExtendedV3[0].Segments[0].Info.Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "BCN",
				Type: 2,
			},
		},
	}

	// TripsExtendedV3[1]
	TripsExtendedV3[1].Price.Currency = &typesv3.Currency{
		Currency: &typesv3.Currency_IsoCurrency{
			IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}

	// TripsExtendedV3[1,0]
	TripsExtendedV3[1].Segments[0].Info.Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "BCN",
				Type: 2,
			},
		},
	}
	TripsExtendedV3[1].Segments[0].Info.Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "MAD",
				Type: 2,
			},
		},
	}

	// TripsExtendedV3[1,1]
	TripsExtendedV3[1].Segments[1].Info.Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "MAD",
				Type: 2,
			},
		},
	}
	TripsExtendedV3[1].Segments[1].Info.Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "LIS",
				Type: 2,
			},
		},
	}

	// TripsExtendedV3[2,0]
	TripsExtendedV3[2].Segments[0].Info.Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "LAN",
				Type: 4,
			},
		},
	}
	TripsExtendedV3[2].Segments[0].Info.Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "HAM",
				Type: 4,
			},
		},
	}

	// TripsExtendedV3[2,1]
	TripsExtendedV3[2].Segments[1].Info.Departure.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "HAM",
				Type: 4,
			},
		},
	}
	TripsExtendedV3[2].Segments[1].Info.Arrival.Location = &transportv3.TransitEventLocation{
		Location: &transportv3.TransitEventLocation_LocationCode{
			LocationCode: &typesv2.LocationCode{
				Code: "BER",
				Type: 4,
			},
		},
	}
	TripsExtendedV3[2].Price.Currency = &typesv3.Currency{
		Currency: &typesv3.Currency_IsoCurrency{
			IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}

	// ActivitySearchV1[0]
	ActivitySearchResultV1[0].Price.Currency = &typesv1.Currency{
		Currency: &typesv1.Currency_IsoCurrency{
			IsoCurrency: typesv1.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}
	// ActivitySearchV1[1]
	ActivitySearchResultV1[1].Price.Currency = &typesv1.Currency{
		Currency: &typesv1.Currency_IsoCurrency{
			IsoCurrency: typesv1.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}

	// ActivitySearchV2[0]
	ActivitySearchResultV2[0].Price.Currency = &typesv2.Currency{
		Currency: &typesv2.Currency_IsoCurrency{
			IsoCurrency: typesv2.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}
	// ActivitySearchV2[1]
	ActivitySearchResultV2[1].Price.Currency = &typesv2.Currency{
		Currency: &typesv2.Currency_IsoCurrency{
			IsoCurrency: typesv2.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}

	// ActivitySearchV3[0]
	ActivitySearchResultV3[0].Price.Currency = &typesv3.Currency{
		Currency: &typesv3.Currency_IsoCurrency{
			IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}
	// ActivitySearchV3[1]
	ActivitySearchResultV3[1].Price.Currency = &typesv3.Currency{
		Currency: &typesv3.Currency_IsoCurrency{
			IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_EUR,
		},
	}
	// ActivitySearchV3[2]
	ActivitySearchResultV3[2].Price.Currency = &typesv3.Currency{
		Currency: &typesv3.Currency_IsoCurrency{
			IsoCurrency: typesv3.IsoCurrency_ISO_CURRENCY_USD,
		},
	}

	// TODO @evlekht do all data checks like make sure that properties has prop.Property.ContactInfo.Address[0] != nil
}

func unmarshalStrictAndValidate[T proto.Message](data []byte) ([]T, error) {
	var raws []json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raws); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}

	var zeroValue T
	messages := make([]T, 0, len(raws))
	for i, raw := range raws {
		typ := reflect.TypeOf(zeroValue).Elem()
		msg := reflect.New(typ).Interface().(T)
		if err := protojson.Unmarshal(raw, msg); err != nil {
			return nil, fmt.Errorf("item %d: protojson unmarshal failed: %w", i, err)
		}
		messages = append(messages, msg)
	}

	for i, item := range messages {
		if err := protovalidate.Validate(item); err != nil {
			return nil, fmt.Errorf("error validating item %d: %w", i, err)
		}
	}

	return messages, nil
}
