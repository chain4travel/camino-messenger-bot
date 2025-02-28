package mockdata

import (
	_ "embed"
	"encoding/json"
	"fmt"

	accommodationv1 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v1"
	accommodationv2 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v2"
	accommodationv3 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/services/accommodation/v3"
)

//go:embed properties.json
var propertiesJSON []byte

var (
	PropertiesV1 []*accommodationv1.PropertyExtendedInfo
	PropertiesV2 []*accommodationv2.PropertyExtendedInfo
	PropertiesV3 []*accommodationv3.PropertyExtendedInfo
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
	// TODO @evlekht do all data checks like make sure that properties has prop.Property.ContactInfo.Address[0] != nil
}
