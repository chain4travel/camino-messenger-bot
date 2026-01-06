package mockdata

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"

	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	accommodationv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v4"
	activityv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v3"
	activityv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v4"

	// activityv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/activity/v5"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	transportv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v4"

	// transportv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v5"
	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	typesv5 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v5"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data/accommodation"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data/activity"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/services/data/transport"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// * Accommodation

//go:embed accommodation/properties_v3.json
var propertiesV3JSON []byte

//go:embed accommodation/properties_v4.json
var propertiesV4JSON []byte

//go:embed accommodation/service_facts_v5.json
var serviceFactsV5JSON []byte

//go:embed accommodation/service_fact_definitions_v5.json
var serviceFactDefinitionsV5JSON []byte

// * Transport

//go:embed transport/trips_v3_basic.json
var tripsV3BasicJSON []byte

//go:embed transport/trips_v3_extended.json
var tripsV3ExtendedJSON []byte

//go:embed transport/trips_v4_basic.json
var tripsV4BasicJSON []byte

//go:embed transport/trips_v4_extended.json
var tripsV4ExtendedJSON []byte

// * Activity

//go:embed activity/activity_v3.json
var activityV3JSON []byte

//go:embed activity/activity_v3_extended.json
var activityV3ExtendedJSON []byte

//go:embed activity/activity_v3_search.json
var activitySearchResultV3JSON []byte

//go:embed activity/activity_v4_extended.json
var activityExtendedV4JSON []byte

//go:embed activity/activity_v4_search.json
var activitySearchResultV4JSON []byte

// * SeatMap

//go:embed seatmap/seatmap_v4.json
var seatMapV4JSON []byte

//go:embed seatmap/seatmap_availability_v4.json
var seatMapAvailabilityV4JSON []byte

var (
	PropertiesV3 []*accommodationv3.PropertyExtendedInfo // used by product list, info and search
	PropertiesV4 []*accommodationv4.PropertyExtendedInfo // used by product list, info and search

	ServiceFactsV5                  []*typesv5.ServiceFact
	ServiceFactDefinitionsV5        []*typesv5.ServiceFactDefinition
	ServiceFactDefinitionsV5Mapping map[string]*typesv5.ServiceFactDefinition

	TripsBasicV3    []*transportv3.TripBasic    // used by product list
	TripsExtendedV3 []*transportv3.TripExtended // used by search

	TripsBasicV4    []*transportv4.TripBasic    // used by product list
	TripsExtendedV4 []*transportv4.TripExtended // used by search
	TripsV4         []*transport.TripV4         // basic+extended, used by search

	ActivityV3             []*activityv3.Activity             // used by product list
	ActivityExtendedV3     []*activityv3.ActivityExtendedInfo // used by product info
	ActivitySearchResultV3 []*activityv3.ActivitySearchResult // used by search

	ActivityExtendedV4     []*activityv4.ActivityExtendedInfo // used by product list and info
	ActivitySearchResultV4 []*activityv4.ActivitySearchResult // used by search

	SeatMapV4             []*typesv4.SeatMap          // used by seatMap
	SeatMapAvailabilityV4 []*typesv4.SeatMapInventory // used by seatMapAvailability
)

func init() {
	// AccommodationV3
	PropertiesV3 = mustUnmarshalStrictAndValidate[*accommodationv3.PropertyExtendedInfo](propertiesV3JSON, "error unmarshaling properties v3")
	// Accommodation V4
	PropertiesV4 = mustUnmarshalStrictAndValidate[*accommodationv4.PropertyExtendedInfo](propertiesV4JSON, "error unmarshaling properties v4")
	// Accommodation V5
	ServiceFactsV5 = mustUnmarshalStrictAndValidate[*typesv5.ServiceFact](serviceFactsV5JSON, "error unmarshaling service facts v5")
	ServiceFactDefinitionsV5 = mustUnmarshalStrictAndValidate[*typesv5.ServiceFactDefinition](serviceFactDefinitionsV5JSON, "error unmarshaling service fact definitions v5")
	ServiceFactDefinitionsV5Mapping = accommodation.VerifyAndGetMapping(ServiceFactsV5, ServiceFactDefinitionsV5)
	// TransportV3
	TripsBasicV3 = mustUnmarshalStrictAndValidate[*transportv3.TripBasic](tripsV3BasicJSON, "error unmarshaling trips basic v3")
	TripsExtendedV3 = mustUnmarshalStrictAndValidate[*transportv3.TripExtended](tripsV3ExtendedJSON, "error unmarshaling trips extended v3")
	// TransportV4
	TripsBasicV4 = mustUnmarshalStrictAndValidate[*transportv4.TripBasic](tripsV4BasicJSON, "error unmarshaling trips basic v4")
	TripsExtendedV4 = mustUnmarshalStrictAndValidate[*transportv4.TripExtended](tripsV4ExtendedJSON, "error unmarshaling trips extended v4")
	TripsV4 = transport.VerifyAndGetTrips(TripsBasicV4, TripsExtendedV4)
	// ActivityV3
	ActivityV3 = mustUnmarshalStrictAndValidate[*activityv3.Activity](activityV3JSON, "error unmarshaling activities v3")
	ActivityExtendedV3 = mustUnmarshalStrictAndValidate[*activityv3.ActivityExtendedInfo](activityV3ExtendedJSON, "error unmarshaling activities extended v3")
	ActivitySearchResultV3 = mustUnmarshalStrictAndValidate[*activityv3.ActivitySearchResult](activitySearchResultV3JSON, "error unmarshaling activities search v3")
	// Activity V4
	ActivityExtendedV4 = mustUnmarshalStrictAndValidate[*activityv4.ActivityExtendedInfo](activityExtendedV4JSON, "error unmarshaling activities extended v4")
	ActivitySearchResultV4 = mustUnmarshalStrictAndValidate[*activityv4.ActivitySearchResult](activitySearchResultV4JSON, "error unmarshaling activities search v4")
	activity.Verify(ActivityExtendedV4, ActivitySearchResultV4)
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
