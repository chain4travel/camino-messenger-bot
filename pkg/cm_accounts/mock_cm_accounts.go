// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts (interfaces: Service)
//
// Generated by this command:
//
//	mockgen -package=cmaccounts -destination=pkg/cm_accounts/mock_cm_accounts.go github.com/chain4travel/camino-messenger-bot/pkg/cm_accounts Service
//

// Package cmaccounts is a generated GoMock package.
package cmaccounts

import (
	context "context"
	ecdsa "crypto/ecdsa"
	big "math/big"
	reflect "reflect"

	cheques "github.com/chain4travel/camino-messenger-bot/pkg/cheques"
	common "github.com/ethereum/go-ethereum/common"
	gomock "go.uber.org/mock/gomock"
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

// CashInCheque mocks base method.
func (m *MockService) CashInCheque(arg0 context.Context, arg1 *cheques.SignedCheque, arg2 *ecdsa.PrivateKey) (common.Hash, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CashInCheque", arg0, arg1, arg2)
	ret0, _ := ret[0].(common.Hash)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CashInCheque indicates an expected call of CashInCheque.
func (mr *MockServiceMockRecorder) CashInCheque(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CashInCheque", reflect.TypeOf((*MockService)(nil).CashInCheque), arg0, arg1, arg2)
}

// GetChequeOperators mocks base method.
func (m *MockService) GetChequeOperators(arg0 context.Context, arg1 common.Address) ([]common.Address, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChequeOperators", arg0, arg1)
	ret0, _ := ret[0].([]common.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetChequeOperators indicates an expected call of GetChequeOperators.
func (mr *MockServiceMockRecorder) GetChequeOperators(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChequeOperators", reflect.TypeOf((*MockService)(nil).GetChequeOperators), arg0, arg1)
}

// GetLastCashIn mocks base method.
func (m *MockService) GetLastCashIn(arg0 context.Context, arg1, arg2, arg3 common.Address) (*big.Int, *big.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastCashIn", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*big.Int)
	ret1, _ := ret[1].(*big.Int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetLastCashIn indicates an expected call of GetLastCashIn.
func (mr *MockServiceMockRecorder) GetLastCashIn(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastCashIn", reflect.TypeOf((*MockService)(nil).GetLastCashIn), arg0, arg1, arg2, arg3)
}

// GetServiceFee mocks base method.
func (m *MockService) GetServiceFee(arg0 context.Context, arg1 common.Address, arg2 string) (*big.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServiceFee", arg0, arg1, arg2)
	ret0, _ := ret[0].(*big.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServiceFee indicates an expected call of GetServiceFee.
func (mr *MockServiceMockRecorder) GetServiceFee(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServiceFee", reflect.TypeOf((*MockService)(nil).GetServiceFee), arg0, arg1, arg2)
}

// IsBotAllowed mocks base method.
func (m *MockService) IsBotAllowed(arg0 context.Context, arg1, arg2 common.Address) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsBotAllowed", arg0, arg1, arg2)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsBotAllowed indicates an expected call of IsBotAllowed.
func (mr *MockServiceMockRecorder) IsBotAllowed(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsBotAllowed", reflect.TypeOf((*MockService)(nil).IsBotAllowed), arg0, arg1, arg2)
}

// VerifyCheque mocks base method.
func (m *MockService) VerifyCheque(arg0 context.Context, arg1 *cheques.SignedCheque) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyCheque", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VerifyCheque indicates an expected call of VerifyCheque.
func (mr *MockServiceMockRecorder) VerifyCheque(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyCheque", reflect.TypeOf((*MockService)(nil).VerifyCheque), arg0, arg1)
}
