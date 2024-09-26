// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/chain4travel/camino-messenger-bot/internal/messaging (interfaces: ServiceRegistry)
//
// Generated by this command:
//
//	mockgen -package=messaging -destination=internal/messaging/mock_service_registry.go github.com/chain4travel/camino-messenger-bot/internal/messaging ServiceRegistry
//

// Package messaging is a generated GoMock package.
package messaging

import (
	reflect "reflect"

	client "github.com/chain4travel/camino-messenger-bot/internal/rpc/client"
	gomock "go.uber.org/mock/gomock"
)

// MockServiceRegistry is a mock of ServiceRegistry interface.
type MockServiceRegistry struct {
	ctrl     *gomock.Controller
	recorder *MockServiceRegistryMockRecorder
}

// MockServiceRegistryMockRecorder is the mock recorder for MockServiceRegistry.
type MockServiceRegistryMockRecorder struct {
	mock *MockServiceRegistry
}

// NewMockServiceRegistry creates a new mock instance.
func NewMockServiceRegistry(ctrl *gomock.Controller) *MockServiceRegistry {
	mock := &MockServiceRegistry{ctrl: ctrl}
	mock.recorder = &MockServiceRegistryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockServiceRegistry) EXPECT() *MockServiceRegistryMockRecorder {
	return m.recorder
}

// GetService mocks base method.
func (m *MockServiceRegistry) GetService(arg0 MessageType) (Service, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetService", arg0)
	ret0, _ := ret[0].(Service)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetService indicates an expected call of GetService.
func (mr *MockServiceRegistryMockRecorder) GetService(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetService", reflect.TypeOf((*MockServiceRegistry)(nil).GetService), arg0)
}

// RegisterServices mocks base method.
func (m *MockServiceRegistry) RegisterServices(arg0 *client.RPCClient) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterServices", arg0)
}

// RegisterServices indicates an expected call of RegisterServices.
func (mr *MockServiceRegistryMockRecorder) RegisterServices(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterServices", reflect.TypeOf((*MockServiceRegistry)(nil).RegisterServices), arg0, arg1)
}
