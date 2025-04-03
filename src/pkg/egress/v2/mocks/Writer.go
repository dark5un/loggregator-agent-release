// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	loggregator_v2 "code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	mock "github.com/stretchr/testify/mock"
)

// Writer is an autogenerated mock type for the Writer type
type Writer struct {
	mock.Mock
}

// Write provides a mock function with given fields: _a0
func (_m *Writer) Write(_a0 *loggregator_v2.Envelope) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Write")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*loggregator_v2.Envelope) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewWriter creates a new instance of Writer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewWriter(t interface {
	mock.TestingT
	Cleanup(func())
}) *Writer {
	mock := &Writer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
