// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package accommodation

import (
	"fmt"

	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
)

func VerifyAndGetMapping(facts []*typesv4.ServiceFact, factDefinitions []*typesv4.ServiceFactDefinition) map[string]*typesv4.ServiceFactDefinition {
	h := &helper{
		definitions:     make(map[string]*typesv4.ServiceFactDefinition),
		factDefinitions: factDefinitions,
	}

	for _, fact := range facts {
		h.setFactDefinition(fact)
	}

	if len(h.definitions) != h.expectedDefinitionsLen {
		panic("mock data error: some service facts have no matching fact definitions")
	}

	return h.definitions
}

type helper struct {
	expectedDefinitionsLen int
	definitions            map[string]*typesv4.ServiceFactDefinition
	factDefinitions        []*typesv4.ServiceFactDefinition
}

func (h *helper) setFactDefinition(fact *typesv4.ServiceFact) {
	h.expectedDefinitionsLen++
	for _, factDefinition := range h.factDefinitions {
		if fact.Code == factDefinition.Code {
			h.definitions[fact.Code] = factDefinition
			for _, subFact := range fact.Details {
				h.setFactDefinition(subFact)
			}
			return
		}
	}
	panic(fmt.Sprintf("mock data error: service fact code %s has no matching fact definition", fact.Code))
}
