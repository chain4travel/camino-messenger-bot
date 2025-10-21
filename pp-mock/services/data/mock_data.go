package mockdata

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"

	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	activityv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v2"
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	transportv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v2"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// * Accommodation

//go:embed accommodation/propertiesv2.json
var propertiesV2JSON []byte

//go:embed accommodation/propertiesv3.json
var propertiesV3JSON []byte

//go:embed accommodation/propertiesv4.json
var propertiesV4JSON []byte

// * Transport

//go:embed transport/tripsv2.json
var tripsV2JSON []byte

//go:embed transport/tripsv3_basic.json
var tripsV3BasicJSON []byte

//go:embed transport/tripsv3_extended.json
var tripsV3ExtendedJSON []byte

// * Activity

//go:embed activity/activityv2.json
var activityV2JSON []byte

//go:embed activity/activityv2_extended.json
var activityExtendedV2JSON []byte

//go:embed activity/activityv2_search.json
var activitySearchResultV2JSON []byte

//go:embed activity/activityv3.json
var activityV3JSON []byte

//go:embed activity/activityv3_extended.json
var activityV3ExtendedJSON []byte

//go:embed activity/activityv3_search.json
var activitySearchResultV3JSON []byte

// * SeatMap

//go:embed seatmap/seatmapv4.json
var seatMapV4JSON []byte

//go:embed seatmap/seatmap_availability_v4.json
var seatMapAvailabilityV4JSON []byte

const (
	SeatMapTransportIndex = 0
	SeatMapActivityIndex  = 1
)

var (
	PropertiesV2 []*accommodationv2.PropertyExtendedInfo // used by product list, info and search
	PropertiesV3 []*accommodationv3.PropertyExtendedInfo // used by product list, info and search
	PropertiesV4 []*accommodationv4.PropertyExtendedInfo // used by product list, info and search

	TripsV2 []*transportv2.Trip // used by search

	TripsBasicV3    []*transportv3.TripBasic    // used by product list
	TripsExtendedV3 []*transportv3.TripExtended // used by search

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
	// AccommodationV2
	PropertiesV2 = mustUnmarshalStrictAndValidate[*accommodationv2.PropertyExtendedInfo](propertiesV2JSON, "error unmarshaling properties v2")
	// AccommodationV3
	PropertiesV3 = mustUnmarshalStrictAndValidate[*accommodationv3.PropertyExtendedInfo](propertiesV3JSON, "error unmarshaling properties v3")
	// Accommodation V4
	PropertiesV4 = mustUnmarshalStrictAndValidate[*accommodationv4.PropertyExtendedInfo](propertiesV4JSON, "error unmarshaling properties v4")
	// TransportV2
	TripsV2 = mustUnmarshalStrictAndValidate[*transportv2.Trip](tripsV2JSON, "error unmarshaling trips v2")
	// TransportV3
	TripsBasicV3 = mustUnmarshalStrictAndValidate[*transportv3.TripBasic](tripsV3BasicJSON, "error unmarshaling trips basic v3")
	TripsExtendedV3 = mustUnmarshalStrictAndValidate[*transportv3.TripExtended](tripsV3ExtendedJSON, "error unmarshaling trips extended v3")
	// ActivityV2
	ActivityV2 = mustUnmarshalStrictAndValidate[*activityv2.Activity](activityV2JSON, "error unmarshaling activities v2")
	ActivityExtendedV2 = mustUnmarshalStrictAndValidate[*activityv2.ActivityExtendedInfo](activityExtendedV2JSON, "error unmarshaling activities extended v2")
	ActivitySearchResultV2 = mustUnmarshalStrictAndValidate[*activityv2.ActivitySearchResult](activitySearchResultV2JSON, "error unmarshaling activities search v2")
	// ActivityV3
	ActivityV3 = mustUnmarshalStrictAndValidate[*activityv3.Activity](activityV3JSON, "error unmarshaling activities v3")
	ActivityExtendedV3 = mustUnmarshalStrictAndValidate[*activityv3.ActivityExtendedInfo](activityV3ExtendedJSON, "error unmarshaling activities extended v3")
	ActivitySearchResultV3 = mustUnmarshalStrictAndValidate[*activityv3.ActivitySearchResult](activitySearchResultV3JSON, "error unmarshaling activities search v3")
	// SeatMapV4
	SeatMapV4 = mustUnmarshalStrictAndValidate[*typesv4.SeatMap](seatMapV4JSON, "error unmarshaling seat map v4")
	SeatMapAvailabilityV4 = mustUnmarshalStrictAndValidate[*typesv4.SeatMapInventory](seatMapAvailabilityV4JSON, "error unmarshaling seat map availability v4")
}

func mustUnmarshalStrictAndValidate[T proto.Message](data []byte, panicMsg string) []T {
	messages, err := unmarshalStrictAndValidate[T](data)
	if err != nil {
		panic(fmt.Errorf("%s: %w", panicMsg, err))
	}
	return messages
}

func unmarshalStrictAndValidate[T proto.Message](data []byte) ([]T, error) {
	var raws []json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raws); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}

	var zeroValue T
	typ := reflect.TypeOf(zeroValue).Elem()

	messages := make([]T, 0, len(raws))
	for i, raw := range raws {
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
