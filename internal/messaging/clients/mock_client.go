// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/chain4travel/camino-messenger-bot/internal/messaging/clients (interfaces: Client)
//
// Generated by this command:
//
//	mockgen -package=clients -destination=internal/messaging/clients/mock_client.go github.com/chain4travel/camino-messenger-bot/internal/messaging/clients Client
//

// Package clients is a generated GoMock package.
package clients

import (
	context "context"
	reflect "reflect"

	messages "github.com/chain4travel/camino-messenger-bot/internal/messaging/messages"
	gomock "go.uber.org/mock/gomock"
	grpc "google.golang.org/grpc"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// Call mocks base method.
func (m *MockClient) Call(arg0 context.Context, arg1 protoreflect.ProtoMessage, arg2 ...grpc.CallOption) (protoreflect.ProtoMessage, messages.MessageType, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Call", varargs...)
	ret0, _ := ret[0].(protoreflect.ProtoMessage)
	ret1, _ := ret[1].(messages.MessageType)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Call indicates an expected call of Call.
func (mr *MockClientMockRecorder) Call(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Call", reflect.TypeOf((*MockClient)(nil).Call), varargs...)
}
