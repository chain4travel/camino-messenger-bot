package server

func (s *server) registerServices() {
	registerPingServiceV1Server(s.grpcServer, s)
	// registerPingServiceV2Server(s.grpcServer, s)
}
