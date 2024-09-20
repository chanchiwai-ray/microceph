// Generated by mockery with a minor update as mockery confuses import paths
package mocks

import (
	context "context"

	state "github.com/canonical/microcluster/v2/state" // mockery gets confused about import paths here
	mock "github.com/stretchr/testify/mock"
)

// MemberCounterInterface is an autogenerated mock type for the MemberCounterInterface type
type MemberCounterInterface struct {
	mock.Mock
}

// Count provides a mock function with given fields: s
func (_m *MemberCounterInterface) Count(ctx context.Context, s state.State) (int, error) {
	ret := _m.Called(s)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, state.State) (int, error)); ok {
		return rf(ctx, s)
	}
	if rf, ok := ret.Get(0).(func(context.Context, state.State) int); ok {
		r0 = rf(ctx, s)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, state.State) error); ok {
		r1 = rf(ctx, s)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CountExclude provides a mock function with given fields: s, exclude
func (_m *MemberCounterInterface) CountExclude(ctx context.Context, s state.State, exclude int64) (int, error) {
	ret := _m.Called(s, exclude)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, state.State, int64) (int, error)); ok {
		return rf(ctx, s, exclude)
	}
	if rf, ok := ret.Get(0).(func(context.Context, state.State, int64) int); ok {
		r0 = rf(ctx, s, exclude)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, state.State, int64) error); ok {
		r1 = rf(ctx, s, exclude)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMemberCounterInterface creates a new instance of MemberCounterInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMemberCounterInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *MemberCounterInterface {
	mock := &MemberCounterInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
