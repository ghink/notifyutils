package errors

import (
	stderrors "errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New("test error message")

	if err.Error() != "test error message" {
		t.Errorf("Error() = %s, want test error message", err.Error())
	}
}

func TestNewWithOptions(t *testing.T) {
	err := New("test message",
		WithDriverName("test-driver"),
		WithDriverCode("ERR001"),
		WithDriverMessage("Driver error message"),
		WithDriverRequestID("req-123"),
		WithDriverResponse(map[string]string{"key": "value"}),
	)

	if err.DriverName() != "test-driver" {
		t.Errorf("DriverName() = %s, want test-driver", err.DriverName())
	}
	if err.DriverCode() != "ERR001" {
		t.Errorf("DriverCode() = %s, want ERR001", err.DriverCode())
	}
	if err.DriverMessage() != "Driver error message" {
		t.Errorf("DriverMessage() = %s, want Driver error message", err.DriverMessage())
	}
	if err.DriverRequestID() != "req-123" {
		t.Errorf("DriverRequestID() = %s, want req-123", err.DriverRequestID())
	}
	if response, ok := err.DriverResponse().(map[string]string); !ok || response["key"] != "value" {
		t.Errorf("DriverResponse() unexpected value: %v", err.DriverResponse())
	}
}

// A decorated error must still match the sentinel it was derived from, which is how
// callers classify failures without knowing the driver.
func TestChainedErrorMatchesSentinel(t *testing.T) {
	err := ErrDriverSendFailed.
		WithDriverName("telegram").
		WithDriverCode("400").
		WithDriverMessage("Bad Request: chat not found")

	if !stderrors.Is(err, ErrDriverSendFailed) {
		t.Errorf("errors.Is() = false, want true for the origin sentinel")
	}
	if stderrors.Is(err, ErrDriverCredentialInvalid) {
		t.Errorf("errors.Is() = true, want false for an unrelated sentinel")
	}
}

// Decoration must not reach back into the sentinel: two sends that fail differently
// cannot leak each other's diagnostics.
func TestSentinelStaysImmutable(t *testing.T) {
	_ = ErrDriverSendFailed.WithDriverName("first").WithDriverCode("500")
	second := ErrDriverSendFailed.WithDriverName("second")

	if ErrDriverSendFailed.DriverName() != "" {
		t.Errorf("sentinel DriverName() = %s, want empty", ErrDriverSendFailed.DriverName())
	}
	if second.DriverName() != "second" {
		t.Errorf("DriverName() = %s, want second", second.DriverName())
	}
	if second.DriverCode() != "" {
		t.Errorf("DriverCode() = %s, want empty on a sibling derivation", second.DriverCode())
	}
}

func TestUnwrapIsNil(t *testing.T) {
	err := ErrDriverSendFailed.WithDriverName("telegram")

	if stderrors.Unwrap(err) != nil {
		t.Errorf("Unwrap() = %v, want nil", stderrors.Unwrap(err))
	}
}

func TestAsFindsTypedError(t *testing.T) {
	err := error(ErrUnsupportedFormat.WithDriverName("email").WithDriverMessage("html"))

	var typed *NotifyutilsError
	if !stderrors.As(err, &typed) {
		t.Fatalf("errors.As() = false, want true")
	}
	if typed.DriverName() != "email" {
		t.Errorf("DriverName() = %s, want email", typed.DriverName())
	}
}
