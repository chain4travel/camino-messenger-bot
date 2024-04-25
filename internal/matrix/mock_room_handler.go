// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/chain4travel/camino-messenger-bot/internal/matrix (interfaces: Client)
//
// Generated by this command:
//
//	mockgen -package=matrix -destination=internal/matrix/mock_room_handler.go github.com/chain4travel/camino-messenger-bot/internal/matrix Client
//

// Package matrix is a generated GoMock package.
package matrix

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	mautrix "maunium.net/go/mautrix"
	event "maunium.net/go/mautrix/event"
	id "maunium.net/go/mautrix/id"
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

// CreateRoom mocks base method.
func (m *MockClient) CreateRoom(arg0 *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRoom", arg0)
	ret0, _ := ret[0].(*mautrix.RespCreateRoom)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRoom indicates an expected call of CreateRoom.
func (mr *MockClientMockRecorder) CreateRoom(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRoom", reflect.TypeOf((*MockClient)(nil).CreateRoom), arg0)
}

// IsEncrypted mocks base method.
func (m *MockClient) IsEncrypted(arg0 id.RoomID) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsEncrypted", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsEncrypted indicates an expected call of IsEncrypted.
func (mr *MockClientMockRecorder) IsEncrypted(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsEncrypted", reflect.TypeOf((*MockClient)(nil).IsEncrypted), arg0)
}

// JoinedMembers mocks base method.
func (m *MockClient) JoinedMembers(arg0 id.RoomID) (*mautrix.RespJoinedMembers, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "JoinedMembers", arg0)
	ret0, _ := ret[0].(*mautrix.RespJoinedMembers)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// JoinedMembers indicates an expected call of JoinedMembers.
func (mr *MockClientMockRecorder) JoinedMembers(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "JoinedMembers", reflect.TypeOf((*MockClient)(nil).JoinedMembers), arg0)
}

// JoinedRooms mocks base method.
func (m *MockClient) JoinedRooms() (*mautrix.RespJoinedRooms, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "JoinedRooms")
	ret0, _ := ret[0].(*mautrix.RespJoinedRooms)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// JoinedRooms indicates an expected call of JoinedRooms.
func (mr *MockClientMockRecorder) JoinedRooms() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "JoinedRooms", reflect.TypeOf((*MockClient)(nil).JoinedRooms))
}

// SendStateEvent mocks base method.
func (m *MockClient) SendStateEvent(arg0 id.RoomID, arg1 event.Type, arg2 string, arg3 any) (*mautrix.RespSendEvent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendStateEvent", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*mautrix.RespSendEvent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendStateEvent indicates an expected call of SendStateEvent.
func (mr *MockClientMockRecorder) SendStateEvent(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendStateEvent", reflect.TypeOf((*MockClient)(nil).SendStateEvent), arg0, arg1, arg2, arg3)
}
