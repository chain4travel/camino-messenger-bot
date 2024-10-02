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

	notificationv1grpc "buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go/cmp/services/notification/v1/notificationv1grpc"
	types "github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	rpc "github.com/chain4travel/camino-messenger-bot/internal/rpc"
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
func (m *MockServiceRegistry) GetService(arg0 types.MessageType) (rpc.Service, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetService", arg0)
	ret0, _ := ret[0].(rpc.Service)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetService indicates an expected call of GetService.
func (mr *MockServiceRegistryMockRecorder) GetService(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetService", reflect.TypeOf((*MockServiceRegistry)(nil).GetService), arg0)
}

// NotificationClient mocks base method.
func (m *MockServiceRegistry) NotificationClient() notificationv1grpc.NotificationServiceClient {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NotificationClient")
	ret0, _ := ret[0].(notificationv1grpc.NotificationServiceClient)
	return ret0
}

// NotificationClient indicates an expected call of NotificationClient.
func (mr *MockServiceRegistryMockRecorder) NotificationClient() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NotificationClient", reflect.TypeOf((*MockServiceRegistry)(nil).NotificationClient))
}
