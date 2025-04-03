// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/ingress/bindings/filtered_binding_fetcher.go

// Package bindings_test is a generated GoMock package.
package bindings_test

import (
	net "net"
	reflect "reflect"

	go_metric_registry "code.cloudfoundry.org/go-metric-registry"
	gomock "github.com/golang/mock/gomock"
)

// MockIPChecker is a mock of IPChecker interface.
type MockIPChecker struct {
	ctrl     *gomock.Controller
	recorder *MockIPCheckerMockRecorder
}

// MockIPCheckerMockRecorder is the mock recorder for MockIPChecker.
type MockIPCheckerMockRecorder struct {
	mock *MockIPChecker
}

// NewMockIPChecker creates a new mock instance.
func NewMockIPChecker(ctrl *gomock.Controller) *MockIPChecker {
	mock := &MockIPChecker{ctrl: ctrl}
	mock.recorder = &MockIPCheckerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIPChecker) EXPECT() *MockIPCheckerMockRecorder {
	return m.recorder
}

// CheckBlacklist mocks base method.
func (m *MockIPChecker) CheckBlacklist(ip net.IP) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckBlacklist", ip)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckBlacklist indicates an expected call of CheckBlacklist.
func (mr *MockIPCheckerMockRecorder) CheckBlacklist(ip interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckBlacklist", reflect.TypeOf((*MockIPChecker)(nil).CheckBlacklist), ip)
}

// ResolveAddr mocks base method.
func (m *MockIPChecker) ResolveAddr(host string) (net.IP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveAddr", host)
	ret0, _ := ret[0].(net.IP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveAddr indicates an expected call of ResolveAddr.
func (mr *MockIPCheckerMockRecorder) ResolveAddr(host interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveAddr", reflect.TypeOf((*MockIPChecker)(nil).ResolveAddr), host)
}

// MockmetricsClient is a mock of metricsClient interface.
type MockmetricsClient struct {
	ctrl     *gomock.Controller
	recorder *MockmetricsClientMockRecorder
}

// MockmetricsClientMockRecorder is the mock recorder for MockmetricsClient.
type MockmetricsClientMockRecorder struct {
	mock *MockmetricsClient
}

// NewMockmetricsClient creates a new mock instance.
func NewMockmetricsClient(ctrl *gomock.Controller) *MockmetricsClient {
	mock := &MockmetricsClient{ctrl: ctrl}
	mock.recorder = &MockmetricsClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockmetricsClient) EXPECT() *MockmetricsClientMockRecorder {
	return m.recorder
}

// NewGauge mocks base method.
func (m *MockmetricsClient) NewGauge(name, helpText string, opts ...go_metric_registry.MetricOption) go_metric_registry.Gauge {
	m.ctrl.T.Helper()
	varargs := []interface{}{name, helpText}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewGauge", varargs...)
	ret0, _ := ret[0].(go_metric_registry.Gauge)
	return ret0
}

// NewGauge indicates an expected call of NewGauge.
func (mr *MockmetricsClientMockRecorder) NewGauge(name, helpText interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{name, helpText}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewGauge", reflect.TypeOf((*MockmetricsClient)(nil).NewGauge), varargs...)
}
