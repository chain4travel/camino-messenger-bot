// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package resources

import (
	"errors"
	"sync"
)

func NewManager(
	minPort int32,
	maxPort int32,
	initialPoolSize int32,
) *Manager {
	availablePorts := make(map[int32]struct{}, initialPoolSize)
	i := minPort
	for ; i < minPort+initialPoolSize && i <= maxPort; i++ {
		availablePorts[i] = struct{}{}
	}
	return &Manager{
		availablePorts: availablePorts,
		nextPort:       i,
		maxPort:        maxPort,
	}
}

// Safe for concurrent use.
type Manager struct {
	mutex          sync.Mutex
	availablePorts map[int32]struct{}
	nextPort       int32
	maxPort        int32
}

func (m *Manager) getNetworkPort() (int32, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch {
	case len(m.availablePorts) == 0 && m.nextPort > m.maxPort:
		return 0, errors.New("no available ports")
	case len(m.availablePorts) == 0:
		port := m.nextPort
		m.nextPort++
		return port, nil
	}

	for port := range m.availablePorts {
		delete(m.availablePorts, port)
		return port, nil
	}

	return 0, errors.New("no available ports")
}

func (m *Manager) releaseNetworkPort(port int32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if port >= m.nextPort {
		return
	}
	m.availablePorts[port] = struct{}{}
}

func (m *Manager) NewSession() *Session {
	return &Session{
		manager: m,
	}
}

// Not safe for concurrent use.
type Session struct {
	manager     *Manager
	lockedPorts []int32
}

func (s *Session) GetNetworkPort() (int32, error) {
	port, err := s.manager.getNetworkPort()
	if err != nil {
		return 0, err
	}
	s.lockedPorts = append(s.lockedPorts, port)
	return port, nil
}

func (s *Session) ReleaseResources() {
	for _, port := range s.lockedPorts {
		s.manager.releaseNetworkPort(port)
	}
}
