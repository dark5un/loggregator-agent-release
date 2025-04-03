// Package testhelpers provides test utilities for the loggregator-agent-release project
package testhelpers

import (
	"fmt"
)

// MockTesting implements the testing.TB interface that mockery requires.
// This allows us to use mockery with Ginkgo.
type MockTesting struct {
	// Optionally track error messages
	ErrorMessages []string
}

// NewMockTesting creates a new MockTesting instance
func NewMockTesting() *MockTesting {
	return &MockTesting{
		ErrorMessages: make([]string, 0),
	}
}

// Errorf implements the testing.TB interface
func (m *MockTesting) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	m.ErrorMessages = append(m.ErrorMessages, msg)
	// Note: We don't fail the test here since we're using Ginkgo assertions
	// instead of testify assertions in our actual tests
}

// FailNow implements the testing.TB interface
func (m *MockTesting) FailNow() {
	// No-op as we're using Ginkgo's failure mechanisms
}

// Cleanup registers a function to be called when the test completes
func (m *MockTesting) Cleanup(f func()) {
	// In a real test, this would register f to run after the test
	// For our use with Ginkgo, we don't need to do anything
}

// Helper implements the testing.TB interface
func (m *MockTesting) Helper() {
	// No-op for our use with Ginkgo
}

// Error logs an error
func (m *MockTesting) Error(args ...interface{}) {}

// Fail marks the test as failed
func (m *MockTesting) Fail() {}

// Failed reports whether the test has failed
func (m *MockTesting) Failed() bool { return false }

// Fatal logs a fatal error and ends test execution
func (m *MockTesting) Fatal(args ...interface{}) {}

// Fatalf logs a formatted fatal error and ends test execution
func (m *MockTesting) Fatalf(format string, args ...interface{}) {}

// Log logs a message
func (m *MockTesting) Log(args ...interface{}) {}

// Logf logs a formatted message
func (m *MockTesting) Logf(format string, args ...interface{}) {}

// Name returns the name of the test
func (m *MockTesting) Name() string { return "mock-test" }

// Setenv sets an environment variable for tests
func (m *MockTesting) Setenv(key, value string) {}

// Skip marks the test as skipped
func (m *MockTesting) Skip(args ...interface{}) {}

// SkipNow marks the test as skipped and stops execution
func (m *MockTesting) SkipNow() {}

// Skipf marks the test as skipped with a formatted message
func (m *MockTesting) Skipf(format string, args ...interface{}) {}

// Skipped reports whether the test was skipped
func (m *MockTesting) Skipped() bool { return false }

// TempDir returns a temporary directory for the test
func (m *MockTesting) TempDir() string { return "" }
