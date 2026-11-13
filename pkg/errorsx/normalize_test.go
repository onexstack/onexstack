package errorsx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantNil     bool
		wantCode    int
		wantReason  string
		wantMessage string
	}{
		{
			name:    "nil stays nil",
			err:     nil,
			wantNil: true,
		},
		{
			name:        "ErrorX passes through",
			err:         ErrNotFound,
			wantCode:    http.StatusNotFound,
			wantReason:  "NotFound",
			wantMessage: ErrNotFound.Message,
		},
		{
			name:        "wrapped ErrorX passes through",
			err:         fmt.Errorf("load row: %w", ErrPermissionDenied),
			wantCode:    http.StatusForbidden,
			wantReason:  "PermissionDenied",
			wantMessage: ErrPermissionDenied.Message,
		},
		{
			name:        "deadline exceeded maps to timeout",
			err:         fmt.Errorf("call upstream: %w", context.DeadlineExceeded),
			wantCode:    http.StatusGatewayTimeout,
			wantReason:  "DeadlineExceeded",
			wantMessage: ErrTimeout.Message,
		},
		{
			name:        "canceled maps to internal",
			err:         context.Canceled,
			wantCode:    http.StatusInternalServerError,
			wantReason:  "InternalError",
			wantMessage: ErrInternal.Message,
		},
		{
			name:        "raw error maps to internal without leaking text",
			err:         errors.New("dial postgres://user:pass@10.0.0.1:5432"),
			wantCode:    http.StatusInternalServerError,
			wantReason:  "InternalError",
			wantMessage: ErrInternal.Message,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("Normalize(nil) = %v, want nil", got)
				}
				return
			}

			var errx *ErrorX
			if !errors.As(got, &errx) {
				t.Fatalf("Normalize() = %T, want *ErrorX", got)
			}
			if errx.Code != tt.wantCode {
				t.Errorf("Code = %d, want %d", errx.Code, tt.wantCode)
			}
			if errx.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", errx.Reason, tt.wantReason)
			}
			if errx.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", errx.Message, tt.wantMessage)
			}
		})
	}
}

func TestNormalize_PreservesCause(t *testing.T) {
	cause := errors.New("boom")
	got := Normalize(fmt.Errorf("outer: %w", cause))

	if !errors.Is(got, cause) {
		t.Fatalf("Normalize() should preserve the cause chain, got %v", got)
	}
}
