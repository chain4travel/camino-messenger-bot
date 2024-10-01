/*
 * Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
 * See the file LICENSE for licensing terms.
 */
package messaging

import (
	"github.com/chain4travel/camino-messenger-bot/internal/messaging/clients/generated"
)

func (s *serviceRegistry) registerServices() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if srv, ok := s.services[generated.PingServiceV1Request]; ok {
		srv.client = generated.NewPingServiceV1(s.rpcClient.ClientConn)
	}
}
