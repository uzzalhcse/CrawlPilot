package driver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uzzalhcse/crawlify/microservices/worker/internal/browser"
)

func TestHttpDriver_NewPage(t *testing.T) {
	d := NewHttpDriver()
	defer d.Close()

	ctx := context.Background()
	page, err := d.NewPage(ctx)
	require.NoError(t, err)
	require.NotNil(t, page)
	require.Equal(t, "http", d.Name())
}

func TestHttpPage_Goto(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<html><head><title>Test Page</title></head><body><div id="content" class="main">Hello World</div><a href="/link">Link</a></body></html>`)
	}))
	defer ts.Close()

	d := NewHttpDriver()
	defer d.Close()

	ctx := context.Background()
	page, err := d.NewPage(ctx)
	require.NoError(t, err)

	// Test Goto success
	err = page.Goto(ts.URL)
	require.NoError(t, err)

	// Test URL
	url, err := page.URL()
	require.NoError(t, err)
	require.Equal(t, ts.URL, url)

	// Test Title
	title, err := page.Title()
	require.NoError(t, err)
	require.Equal(t, "Test Page", title)

	// Test Content
	content, err := page.Content()
	require.NoError(t, err)
	require.Contains(t, content, "Hello World")
}

func TestHttpPage_Goto_Error(t *testing.T) {
	d := NewHttpDriver()
	defer d.Close()

	ctx := context.Background()
	page, err := d.NewPage(ctx)
	require.NoError(t, err)

	// Test invalid URL
	err = page.Goto("http://invalid-url-that-does-not-exist.local")
	require.Error(t, err)
}

func TestHttpPage_Selectors(t *testing.T) {
	html := `
		<html>
			<body>
				<div id="d1" data-test="value1">Div 1</div>
				<div class="item">Item 1</div>
				<div class="item">Item 2</div>
				<span id="empty"></span>
			</body>
		</html>
	`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}))
	defer ts.Close()

	d := NewHttpDriver()
	defer d.Close()

	page, _ := d.NewPage(context.Background())
	require.NoError(t, page.Goto(ts.URL))

	// QuerySelector
	el, err := page.QuerySelector("#d1")
	require.NoError(t, err)
	require.NotNil(t, el)

	text, err := el.Text()
	require.NoError(t, err)
	require.Equal(t, "Div 1", text)

	attr, err := el.Attribute("data-test")
	require.NoError(t, err)
	require.Equal(t, "value1", attr)

	// QuerySelectorAll
	els, err := page.QuerySelectorAll(".item")
	require.NoError(t, err)
	require.Len(t, els, 2)

	text1, _ := els[0].Text()
	text2, _ := els[1].Text()
	require.Equal(t, "Item 1", text1)
	require.Equal(t, "Item 2", text2)

	// QuerySelector (not found)
	el, err = page.QuerySelector("#not-found")
	require.NoError(t, err)
	require.Nil(t, el)
}

func TestHttpPage_WaitForSelector(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><div id="exist"></div></body></html>`)
	}))
	defer ts.Close()

	d := NewHttpDriver()
	page, _ := d.NewPage(context.Background())
	page.Goto(ts.URL)

	// Exists
	err := page.WaitForSelector("#exist")
	require.NoError(t, err)

	// Does not exist (should fail immediately for HTTP driver as it's static)
	err = page.WaitForSelector("#not-exist", WithWaitTimeout(100*time.Millisecond))
	require.ErrorIs(t, err, ErrElementNotFound)

	// Wait for hidden/detached (should succeed if not present)
	err = page.WaitForSelector("#not-exist", WithState("hidden"))
	require.NoError(t, err)
}

func TestHttpPage_UnsupportedMethods(t *testing.T) {
	d := NewHttpDriver()
	page, _ := d.NewPage(context.Background())

	require.ErrorIs(t, page.Click("#id"), ErrNotSupported)
	require.ErrorIs(t, page.Type("#id", "text"), ErrNotSupported)
	require.ErrorIs(t, page.Fill("#id", "text"), ErrNotSupported)
	require.ErrorIs(t, page.Hover("#id"), ErrNotSupported)

	_, err := page.Screenshot()
	require.ErrorIs(t, err, ErrNotSupported)

	_, err = page.Evaluate("console.log('test')")
	require.ErrorIs(t, err, ErrNotSupported)
}

func TestHttpPage_Proxy(t *testing.T) {
	// Start a mock proxy server
	proxyCalled := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyCalled = true
		// Forward request (simplified for test)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Proxy Response"))
	}))
	defer proxy.Close()

	d := NewHttpDriver()
	defer d.Close()

	// Create context with proxy config
	proxyCfg := &browser.ProxyConfig{
		Server: proxy.URL,
	}
	ctx := context.WithValue(context.Background(), ProxyKey, proxyCfg)

	page, err := d.NewPage(ctx)
	require.NoError(t, err)

	// Make request (destination doesn't matter as proxy intercepts)
	err = page.Goto("http://example.com")
	require.NoError(t, err)

	// Verify proxy was used
	assert.True(t, proxyCalled, "Proxy should have been called")

	content, _ := page.Content()
	assert.Contains(t, content, "Proxy Response")
}
