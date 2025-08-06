package mockdata

import (
	"encoding/json"
	"fmt"
	"testing"

	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/sryoya/protorand"
	"github.com/stretchr/testify/require"
)

// Utility to generate random JSON data for a specific proto message type
func TestGenerateRandomJSON(t *testing.T) {
	// t.Skip() // comment this line to run util
	pr := protorand.New()

	protoMessageType := &typesv4.SeatMapInventory{} // set the type of the proto message you want to generate
	pr.MaxCollectionElements = 1                    // set the maximum number of elements in arrays

	fakeProtoMessage, err := pr.Gen(protoMessageType)
	require.NoError(t, err)
	propertiesJSON, err := json.Marshal(fakeProtoMessage)
	require.NoError(t, err)
	fmt.Println(string(propertiesJSON))
}

// will run init() function to load mock data
func TestMockData(*testing.T) {}
