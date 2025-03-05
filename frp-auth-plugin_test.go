package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyu-rts/frp-auth-plugin/internal/configfile"
)

type mockConfigProvider struct {
	config configfile.Config
}

func (p mockConfigProvider) CurrentConfig() *configfile.Config {
	return &p.config
}

func TestInteg(t *testing.T) {
	config := configfile.Config{
		Users: map[string]configfile.User{
			"user1": configfile.User{
				Password: "hackme",
				Proxies: map[string]configfile.Proxy{
					"proxy1": configfile.Proxy{
						CustomDomains: []string{
							"user1-a.example.org",
							"user1-b.example.org",
						},
						HttpUser:     "http1",
						HttpPassword: "password",
					},
				},
			},
		},
	}

	// Set config, create server
	ConfigFile = mockConfigProvider{
		config,
	}
	server := httptest.NewServer(http.HandlerFunc(handleReq))
	defer server.Close()
	client := server.Client()

	// Test Login request
	res, err := client.Post(server.URL+"/handler?op=Login", "application/json", bytes.NewBufferString("{\"content\": {\"user\": \"user1\", \"metas\": {\"token\": \"hackme\"}}}"))
	if err != nil {
		t.Errorf("error sending request: %v", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("error reading body: %v", err)
	}
	if !bytes.Equal(data, []byte("{\"reject\": false, \"unchange\": true}")) {
		t.Errorf("invalid response: %v", string(data))
	}
}
