package mockdata

import (
	_ "embed"
	"encoding/json"
	"fmt"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
	transportv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v1"
	transportv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v2"
	transportv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/transport/v3"
	typesv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v2"
	typesv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v3"
)

//go:embed properties.json
var propertiesJSON []byte

//go:embed tripsv1.json
var tripsV1JSON []byte

//go:embed tripsv3_basic.json
var tripsV3BasicJSON []byte

//go:embed tripsv3_extended.json
var tripsV3ExtendedJSON []byte

var (
	PropertiesV1 []*accommodationv1.PropertyExtendedInfo
	PropertiesV2 []*accommodationv2.PropertyExtendedInfo
	PropertiesV3 []*accommodationv3.PropertyExtendedInfo

	TripsV1 []*transportv1.Trip
	TripsV2 []*transportv2.Trip

	TripsBasicV3    []*transportv3.TripBasic
	TripsExtendedV3 []*transportv3.TripExtended

	LocationCodeToAmountPerDay = map[string]float32{
		"PMI": 25.50,
		"BCN": 30.75,
		"MAD": 28.25,
		"LIS": 22.10,
		"LAN": 35.00,
		"HAM": 40.50,
		"BER": 38.75,
	}

	// Transport carbon compensation amounts per route (kg CO2)
	TransportRouteToAmount = map[string]float32{
		"HAM-BER": 45.20,
		"BER-HAM": 45.20,
		"PMI-BCN": 52.30,
		"BCN-PMI": 52.30,
		"BCN-MAD": 38.90,
		"MAD-BCN": 38.90,
		"MAD-LIS": 42.15,
		"LIS-MAD": 42.15,
		"LAN-HAM": 65.80,
		"HAM-LAN": 65.80,
	}

	// Plane transport carbon compensation (kg CO2) - higher emissions
	PlaneRouteToAmount = map[string]float32{
		"HAM-BER": 45.20,
		"BER-HAM": 45.20,
		"PMI-BCN": 52.30,
		"BCN-PMI": 52.30,
		"BCN-MAD": 38.90,
		"MAD-BCN": 38.90,
		"MAD-LIS": 42.15,
		"LIS-MAD": 42.15,
		"LAN-HAM": 65.80,
		"HAM-LAN": 65.80,
		"PMI-MAD": 68.50,
		"MAD-PMI": 68.50,
		"LIS-BCN": 55.40,
		"BCN-LIS": 55.40,
	}

	// Train transport carbon compensation (kg CO2) - much lower emissions
	TrainRouteToAmount = map[string]float32{
		"HAM-BER": 4.10,
		"BER-HAM": 4.10,
		"BCN-MAD": 2.90,
		"MAD-BCN": 2.90,
		"MAD-LIS": 3.10,
		"LIS-MAD": 3.10,
		"HAM-LAN": 6.15,
		"LAN-HAM": 6.15,
		"BER-LAN": 5.20,
		"LAN-BER": 5.20,
	}
)

func init() {
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

	// because protobuf location and price are one-of interface types,
	// json unmarshaling won't work for them and will result in error
	// so, as quick workaround, we are setting them manually

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
	// TODO @evlekht do all data checks like make sure that properties has prop.Property.ContactInfo.Address[0] != nil
}
