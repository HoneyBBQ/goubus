package goubus_test

import (
	"errors"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func TestCallErrorUnwrapTree(t *testing.T) {
	callErr := &goubus.CallError{
		Err:     goubus.ErrNotFound,
		Service: "system",
		Method:  "info",
		Message: "missing",
		Code:    goubus.StatusNotFound,
	}
	if got, want := callErr.Error(), "ubus call system.info failed (code 4: missing): not found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(callErr, goubus.ErrCall) || !errors.Is(callErr, goubus.ErrNotFound) {
		t.Errorf("Unwrap() error tree does not contain both sentinels: %v", callErr)
	}
	got := callErr.Unwrap()
	if len(got) != 2 || !errors.Is(got[0], goubus.ErrCall) || !errors.Is(got[1], goubus.ErrNotFound) {
		t.Errorf("Unwrap() = %#v, want ErrCall and ErrNotFound", got)
	}
	got = (&goubus.CallError{Code: 99}).Unwrap()
	if len(got) != 1 || !errors.Is(got[0], goubus.ErrCall) {
		t.Errorf("Unwrap() without cause = %#v, want ErrCall", got)
	}
	if got := (&goubus.CallError{Method: "list", Code: 1}).Error(); got != "ubus list failed (code 1): <nil>" {
		t.Errorf("Error() without service = %q", got)
	}
	withMsg := (&goubus.CallError{Method: "list", Message: "denied", Code: 6}).Error()
	if withMsg != "ubus list failed (code 6: denied): <nil>" {
		t.Errorf("Error() without service with message = %q", withMsg)
	}
	noMsg := (&goubus.CallError{Service: "uci", Method: "get", Err: goubus.ErrNotFound, Code: 4}).Error()
	if noMsg != "ubus call uci.get failed (code 4): not found" {
		t.Errorf("Error() without message = %q", noMsg)
	}
}

func TestStatusErrorMapping(t *testing.T) {
	if goubus.StatusError(goubus.StatusOK) != nil {
		t.Fatalf("StatusError(StatusOK) = %v, want nil", goubus.StatusError(goubus.StatusOK))
	}
	for code, want := range map[int]error{
		goubus.StatusNotFound:    goubus.ErrNotFound,
		goubus.StatusNoMemory:    goubus.ErrNoMemory,
		goubus.StatusParseError:  goubus.ErrParseError,
		goubus.StatusSystemError: goubus.ErrSystemError,
		999:                      goubus.ErrUnknown,
		-1:                       goubus.ErrUnknown,
	} {
		if !errors.Is(goubus.StatusError(code), want) {
			t.Errorf("StatusError(%d) did not match %v", code, want)
		}
	}
}
