// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package accommodation

import (
	"fmt"

	typesv4 "buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go/cmp/types/v4"
	"github.com/chain4travel/camino-messenger-bot/v12/pp-mock/common"
)

type ServiceV4 struct {
	Fact           *typesv4.ServiceFact
	FactDefinition *typesv4.ServiceFactDefinition
}

func (t *ServiceV4) Clone() *ServiceV4 {
	return &ServiceV4{
		Fact:           common.CloneProto(t.Fact),
		FactDefinition: common.CloneProto(t.FactDefinition),
	}
}

func (t *ServiceV4) Verify() error {
	if t.Fact.Code != t.FactDefinition.Code {
		return fmt.Errorf("trip basic and extended supplier code mismatch")
	}
	return nil
}

func VerifyAndGetServices(facts []*typesv4.ServiceFact, factDefinitions []*typesv4.ServiceFactDefinition) []*ServiceV4 {
	if len(facts) != len(factDefinitions) {
		panic(fmt.Errorf("mock data error: number of serviceFact (%d) does not match number of serviceFactDefinition (%d)", len(facts), len(factDefinitions)))
	}
	services := make([]*ServiceV4, 0, len(facts))
	for i, fact := range facts {
		service := &ServiceV4{
			Fact:           fact,
			FactDefinition: factDefinitions[i],
		}
		if err := service.Verify(); err != nil {
			panic(fmt.Errorf("mock data error: service fact/fact definition at index %d are invalid: %w", i, err))
		}
		services = append(services, service)
	}
	return services
}

func CloneV4(services []*ServiceV4) []*ServiceV4 {
	cloned := make([]*ServiceV4, len(services))
	for i, service := range services {
		cloned[i] = service.Clone()
	}
	return cloned
}

func FactV4(services []*ServiceV4) []*typesv4.ServiceFact {
	facts := make([]*typesv4.ServiceFact, len(services))
	for i, service := range services {
		facts[i] = common.CloneProto(service.Fact)
	}
	return facts
}

func FactDefinitionV4(services []*ServiceV4) []*typesv4.ServiceFactDefinition {
	factDefinitions := make([]*typesv4.ServiceFactDefinition, len(services))
	for i, service := range services {
		factDefinitions[i] = common.CloneProto(service.FactDefinition)
	}
	return factDefinitions
}
