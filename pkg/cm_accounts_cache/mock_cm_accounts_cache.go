// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts_cache (interfaces: CMAccountsCache)
//
// Generated by this command:
//
//	mockgen -package=cmaccountscache -destination=pkg/cm_accounts_cache/mock_cm_accounts_cache.go github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts_cache CMAccountsCache
//

// Package cmaccountscache is a generated GoMock package.
package cmaccountscache

import (
	reflect "reflect"

	cmaccount "github.com/chain4travel/camino-messenger-contracts/go/contracts/cmaccount"
	common "github.com/ethereum/go-ethereum/common"
	gomock "go.uber.org/mock/gomock"
)

// MockCMAccountsCache is a mock of CMAccountsCache interface.
type MockCMAccountsCache struct {
	ctrl     *gomock.Controller
	recorder *MockCMAccountsCacheMockRecorder
}

// MockCMAccountsCacheMockRecorder is the mock recorder for MockCMAccountsCache.
type MockCMAccountsCacheMockRecorder struct {
	mock *MockCMAccountsCache
}

// NewMockCMAccountsCache creates a new mock instance.
func NewMockCMAccountsCache(ctrl *gomock.Controller) *MockCMAccountsCache {
	mock := &MockCMAccountsCache{ctrl: ctrl}
	mock.recorder = &MockCMAccountsCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCMAccountsCache) EXPECT() *MockCMAccountsCacheMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockCMAccountsCache) Get(arg0 common.Address) (*cmaccount.Cmaccount, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(*cmaccount.Cmaccount)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockCMAccountsCacheMockRecorder) Get(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCMAccountsCache)(nil).Get), arg0)
}