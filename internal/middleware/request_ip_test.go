package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ipProbeApp answers GET /whoami with the IP that c.IP() reports.
//
// The 0.0.0.0/8 entry exists because app.Test presents the peer as 0.0.0.0:
// without trusting that range no request would ever count as coming from a
// trusted proxy in these tests.
func ipProbeApp(trusted bool) *fiber.App {
	cfg := fiber.Config{}
	if trusted {
		cfg.TrustProxy = true
		cfg.TrustProxyConfig = fiber.TrustProxyConfig{
			Loopback: true,
			Proxies:  []string{"0.0.0.0/8", "10.0.0.0/8"},
		}
		cfg.ProxyHeader = fiber.HeaderXForwardedFor
		cfg.EnableIPValidation = true
	}

	app := fiber.New(cfg)
	app.Get("/whoami", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"ip": c.IP()})
	})
	return app
}

func whoami(t *testing.T, app *fiber.App, forwardedFor string) string {
	t.Helper()

	req := httptest.NewRequest(fiber.MethodGet, "/whoami", nil)
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}

	res, err := app.Test(req, fiber.TestConfig{Timeout: 10000})
	require.NoError(t, err)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	var body struct {
		IP string `json:"ip"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	return body.IP
}

func TestClientIPTrustedProxyExtractsRealClient(t *testing.T) {
	tests := []struct {
		name         string
		forwardedFor string
		want         string
	}{
		{
			name:         "single client behind one proxy",
			forwardedFor: "203.0.113.42",
			want:         "203.0.113.42",
		},
		{
			name:         "multi hop chain returns the leftmost real client",
			forwardedFor: "203.0.113.42, 10.1.2.3, 127.0.0.1",
			want:         "203.0.113.42",
		},
		{
			name:         "rightmost untrusted wins when chain ends outside trust",
			forwardedFor: "198.51.100.9, 203.0.113.42",
			want:         "203.0.113.42",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, whoami(t, ipProbeApp(true), tc.forwardedFor))
		})
	}
}

func TestClientIPSpoofingIgnoredFromUntrustedPeer(t *testing.T) {
	// The peer is loopback but not trusted (TrustProxy off), so a forged
	// X-Forwarded-For must never be believed.
	app := ipProbeApp(false)

	got := whoami(t, app, "203.0.113.42")
	assert.NotEqual(t, "203.0.113.42", got, "an untrusted peer must not be able to spoof its source IP")
	assert.Contains(t, []string{"127.0.0.1", "::1", "0.0.0.0"}, got, "must fall back to the TCP peer")
}

func TestClientIPWithoutHeader(t *testing.T) {
	app := ipProbeApp(true)

	got := whoami(t, app, "")
	assert.Contains(t, []string{"127.0.0.1", "::1", "0.0.0.0"}, got,
		"no header means the direct peer is all we know")
}

func TestClientIPIgnoresGarbageEntries(t *testing.T) {
	app := ipProbeApp(true)

	// "not-an-ip" is not a valid address; validation must skip junk and still
	// find the real client at the left.
	got := whoami(t, app, "203.0.113.42, not-an-ip, ")
	assert.Equal(t, "203.0.113.42", got)
}
