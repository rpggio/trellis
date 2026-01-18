package transport

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRequest(t *testing.T) {
	body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"test","params":{"a":1},"id":1}`)
	req, err := ParseRequest(body)
	require.NoError(t, err)
	require.Equal(t, "2.0", req.JSONRPC)
	require.Equal(t, "test", req.Method)
	require.Equal(t, json.RawMessage(`{"a":1}`), req.Params)
}

func TestParseRequest_Invalid(t *testing.T) {
	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1}`)
	_, err := ParseRequest(body)
	require.Error(t, err)
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, 1, ErrInvalidParams, "bad params", nil)

	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `"error"`)
}
