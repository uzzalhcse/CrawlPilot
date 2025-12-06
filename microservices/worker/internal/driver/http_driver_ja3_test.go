package driver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpPage_JA3(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("JA3 Test"))
	}))
	defer ts.Close()

	d := NewHttpDriver()
	defer d.Close()

	// Create context with JA3 config
	ja3Cfg := &JA3Config{
		BrowserName: "chrome",
	}
	ctx := context.WithValue(context.Background(), JA3Key, ja3Cfg)

	page, err := d.NewPage(ctx)
	require.NoError(t, err)

	// Make request
	err = page.Goto(ts.URL)
	require.NoError(t, err)

	content, err := page.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "JA3 Test")
}
