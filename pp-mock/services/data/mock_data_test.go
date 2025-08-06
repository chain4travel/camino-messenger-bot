package mockdata

import (
	"testing"

	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/sryoya/protorand"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

// Utility to generate random JSON data for a specific proto message type
func TestGenerateRandomJSON(t *testing.T) {
	t.Skip() // comment this line to run util
	protorand := protorand.New()

	protoMessageType := &typesv4.SeatMapInventory{} // set the type of the proto message you want to generate
	protorand.MaxCollectionElements = 1             // set the maximum number of elements in arrays

	fakeProtoMessage, err := protorand.Gen(protoMessageType)
	require.NoError(t, err)
	println(protojson.Format(fakeProtoMessage))
}

// will run init() function to load mock data
func TestMockData(*testing.T) {}
