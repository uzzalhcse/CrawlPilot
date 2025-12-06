package driver

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
)

// UTLSRoundTripper implements http.RoundTripper using utls to mimic browser TLS handshakes
type UTLSRoundTripper struct {
	clientHelloID utls.ClientHelloID
	proxyURL      *url.URL
	insecure      bool
}

// NewUTLSTransport creates an http.Transport that uses utls for TLS connections
func NewUTLSTransport(clientHelloID utls.ClientHelloID, proxyURL *url.URL, insecure bool) *http.Transport {
	return &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return proxyURL, nil
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 1. Establish underlying TCP connection
			var conn net.Conn
			var err error

			if proxyURL != nil {
				conn, err = dialViaProxy(ctx, proxyURL, addr)
			} else {
				dialer := net.Dialer{Timeout: 30 * time.Second}
				conn, err = dialer.DialContext(ctx, network, addr)
			}

			if err != nil {
				return nil, err
			}

			// 2. Create utls connection
			host, _, _ := net.SplitHostPort(addr)
			uConn := utls.UClient(conn, &utls.Config{
				ServerName:         host,
				InsecureSkipVerify: insecure,
			}, clientHelloID)

			// 3. Build handshake state first, then modify ALPN
			if err := uConn.BuildHandshakeState(); err != nil {
				conn.Close()
				return nil, fmt.Errorf("failed to build handshake state: %w", err)
			}

			// 4. Force HTTP/1.1 by modifying the ALPN extension
			// Find and modify ALPN extension to only contain http/1.1
			for _, ext := range uConn.Extensions {
				if alpnExt, ok := ext.(*utls.ALPNExtension); ok {
					alpnExt.AlpnProtocols = []string{"http/1.1"}
					break
				}
			}

			// 5. Perform handshake with modified extensions
			if err := uConn.Handshake(); err != nil {
				conn.Close()
				return nil, fmt.Errorf("utls handshake failed: %w", err)
			}

			return uConn, nil
		},
		ForceAttemptHTTP2: false,
	}
}

// dialViaProxy establishes a connection to the target via the proxy
func dialViaProxy(ctx context.Context, proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	// Connect to proxy
	proxyAddr := proxyURL.Host
	if !strings.Contains(proxyAddr, ":") {
		proxyAddr += ":80" // Default proxy port
	}

	dialer := net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	// Send CONNECT request
	connectReq := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: targetAddr},
		Host:   targetAddr,
		Header: make(http.Header),
	}

	// Add Proxy-Authorization if needed
	if user := proxyURL.User; user != nil {
		password, _ := user.Password()
		auth := user.Username() + ":" + password
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		connectReq.Header.Set("Proxy-Authorization", basicAuth)
	}

	if err := connectReq.Write(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to write CONNECT request: %w", err)
	}

	// Read response
	// Minimal check: read until double newline or first line
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
	}

	respStr := string(buf[:n])
	if !strings.Contains(respStr, "200 Connection established") && !strings.Contains(respStr, "200 OK") {
		conn.Close()
		return nil, fmt.Errorf("proxy refused connection: %s", strings.Split(respStr, "\n")[0])
	}

	return conn, nil
}
