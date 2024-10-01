// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/chain4travel/camino-messenger-bot/internal/messaging (interfaces: Service)
//
// Generated by this command:
//
//	mockgen -package=messaging -destination=internal/messaging/mock_service.go github.com/chain4travel/camino-messenger-bot/internal/messaging Service
//

// Package messaging is a generated GoMock package.
package messaging

import (
	context "context"
	reflect "reflect"

	types "github.com/chain4travel/camino-messenger-bot/internal/messaging/types"
	gomock "go.uber.org/mock/gomock"
	grpc "google.golang.org/grpc"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

// MockService is a mock of Service interface.
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
}

// MockServiceMockRecorder is the mock recorder for MockService.
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance.
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// Call mocks base method.
func (m *MockService) Call(arg0 context.Context, arg1 protoreflect.ProtoMessage, arg2 ...grpc.CallOption) (protoreflect.ProtoMessage, types.MessageType, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Call", varargs...)
	ret0, _ := ret[0].(protoreflect.ProtoMessage)
	ret1, _ := ret[1].(types.MessageType)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Call indicates an expected call of Call.
func (mr *MockServiceMockRecorder) Call(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Call", reflect.TypeOf((*MockService)(nil).Call), varargs...)
}

// Name mocks base method.
func (m *MockService) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockServiceMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockService)(nil).Name))
}
