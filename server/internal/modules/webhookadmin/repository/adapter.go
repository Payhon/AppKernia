package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	webhooks "github.com/appkernia/appkernia/server/internal/modules/webhookadmin/domain"
)

type LocalMockAdapter struct{}

func NewLocalMockAdapter() *LocalMockAdapter { return &LocalMockAdapter{} }
func (a *LocalMockAdapter) Deliver(_ context.Context, raw string, headers map[string]string, body []byte, _ time.Duration) (webhooks.DeliveryResult, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return webhooks.DeliveryResult{}, webhooks.ErrDelivery
	}
	host := strings.ToLower(u.Hostname())
	if host != "example.test" && !strings.HasSuffix(host, ".example.test") {
		return webhooks.DeliveryResult{}, errors.New("local mock only accepts example.test hosts")
	}
	if headers["X-AK-Webhook-Signature"] == "" || headers["X-AK-Webhook-Timestamp"] == "" || headers["X-AK-Webhook-Event-ID"] == "" || len(body) == 0 {
		return webhooks.DeliveryResult{}, errors.New("webhook signature envelope missing")
	}
	return webhooks.DeliveryResult{StatusCode: http.StatusNoContent, Body: ""}, nil
}

type HTTPAdapter struct{ resolver *net.Resolver }

func NewHTTPAdapter() *HTTPAdapter { return &HTTPAdapter{resolver: net.DefaultResolver} }
func public(ip netip.Addr) bool {
	return ip.IsValid() && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
func (a *HTTPAdapter) Deliver(ctx context.Context, raw string, headers map[string]string, body []byte, timeout time.Duration) (webhooks.DeliveryResult, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return webhooks.DeliveryResult{}, webhooks.ErrDelivery
	}
	addresses, err := a.resolver.LookupNetIP(ctx, "ip", u.Hostname())
	if err != nil || len(addresses) == 0 {
		return webhooks.DeliveryResult{}, fmt.Errorf("resolve webhook host: %w", err)
	}
	for _, ip := range addresses {
		if !public(ip) {
			return webhooks.DeliveryResult{}, errors.New("webhook host resolved to a non-public address")
		}
	}
	chosen := addresses[0]
	port := u.Port()
	if port == "" {
		port = "443"
	}
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, net.JoinHostPort(chosen.String(), port))
	}, TLSHandshakeTimeout: timeout, ForceAttemptHTTP2: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("webhook redirects are disabled") }}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, raw, strings.NewReader(string(body)))
	if err != nil {
		return webhooks.DeliveryResult{}, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return webhooks.DeliveryResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	rawBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4097))
	if readErr != nil {
		return webhooks.DeliveryResult{}, readErr
	}
	if len(rawBody) > 4000 {
		rawBody = rawBody[:4000]
	}
	result := webhooks.DeliveryResult{StatusCode: resp.StatusCode, Body: string(rawBody)}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return result, fmt.Errorf("webhook receiver returned HTTP %d", resp.StatusCode)
	}
	return result, nil
}
