package webhook

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JetBrains/air-goland-demo/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readEvent(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(filepath.Join("testdata", "events", name))
	require.NoError(t, err)
	return body
}

func post(h http.Handler, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(body))
	req.SetPathValue("provider", "stripe")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	t.Run("New event", func(t *testing.T) {
		t.Parallel()

		h := NewHandler(store.NewMemory())
		rec := post(h, readEvent(t, "payment_succeeded.json"))
		assert.Equal(t, http.StatusAccepted, rec.Code)
	})

	t.Run("Duplicate event", func(t *testing.T) {
		t.Parallel()

		h := NewHandler(store.NewMemory())
		body := readEvent(t, "payment_succeeded.json")
		post(h, body)
		rec := post(h, body)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Invalid event ID", func(t *testing.T) {
		t.Parallel()

		h := NewHandler(store.NewMemory())
		rec := post(h, readEvent(t, "invalid_id.json"))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "invalid event id", strings.TrimSpace(rec.Body.String()))
	})

	t.Run("Body too large", func(t *testing.T) {
		t.Parallel()

		h := NewHandler(store.NewMemory())
		rec := post(h, []byte(`{"id": "`+strings.Repeat("a", maxBodyBytes)+`"}`))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "invalid JSON body", strings.TrimSpace(rec.Body.String()))
	})
}

func BenchmarkHandler(b *testing.B) {
	h := NewHandler(store.NewMemory())
	payloads := make([][]byte, b.N)
	for i := range payloads {
		payloads[i] = fmt.Appendf(nil,
			`{"id": "evt_%024d", "type": "payment.succeeded", "data": {"amount": 2500, "currency": "gbp"}}`, i)
	}

	b.ResetTimer()
	for i := range b.N {
		post(h, payloads[i])
	}
}
