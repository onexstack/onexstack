package core

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/onexstack/onexstack/pkg/errorsx"
)

func newJSONRecorder() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	return c, recorder
}

func decodeErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()

	var body ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body %q: %v", recorder.Body.String(), err)
	}

	return body
}

func TestWriteResponse_RawErrorIsNormalized(t *testing.T) {
	c, recorder := newJSONRecorder()

	WriteResponse(c, nil, errors.New("dial postgres://user:pass@10.0.0.1:5432"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	body := decodeErrorResponse(t, recorder)
	if body.Reason != errorsx.ErrInternal.Reason {
		t.Errorf("reason = %q, want %q", body.Reason, errorsx.ErrInternal.Reason)
	}
	if body.Message != errorsx.ErrInternal.Message {
		t.Errorf("message = %q, want %q", body.Message, errorsx.ErrInternal.Message)
	}
	if strings.Contains(recorder.Body.String(), "postgres://") {
		t.Errorf("response leaked the raw error text: %s", recorder.Body.String())
	}
}

func TestWriteResponse_ErrorXKeepsStatusAndReason(t *testing.T) {
	c, recorder := newJSONRecorder()

	WriteResponse(c, nil, errorsx.ErrNotFound)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	body := decodeErrorResponse(t, recorder)
	if body.Reason != errorsx.ErrNotFound.Reason {
		t.Errorf("reason = %q, want %q", body.Reason, errorsx.ErrNotFound.Reason)
	}
}

func TestWriteResponse_SuccessIsUntouched(t *testing.T) {
	c, recorder := newJSONRecorder()

	WriteResponse(c, map[string]string{"status": "ok"}, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); body != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", body, `{"status":"ok"}`)
	}
}
