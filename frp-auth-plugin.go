package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/remram44/frp-auth-plugin/internal/configfile"
)

var config *configfile.Config

func main() {
	ctx := context.Background()

	// Read command line
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: frp-auth-plugin <configuration-file>\n")
		os.Exit(2)
	}
	configFile := os.Args[1]

	// Read configuration
	config, err := configfile.New(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %s\n", err)
		os.Exit(1)
	}

	// Read listen address
	listenAddr := os.Getenv("FRP_AUTH_PLUGIN_LISTEN_ADDR")
	if listenAddr == "" {
		fmt.Fprintf(os.Stderr, "FRP_AUTH_PLUGIN_LISTEN_ADDR is not set\n")
		os.Exit(2)
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("POST /handler", handleReq)
	server := http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}
	context.AfterFunc(ctx, func() { server.Close() })
	log.Printf("Listening on %s", listenAddr)
	err = server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	_ = config
}

type FrpLogin struct {
	User          string            `json:"user"`
	PrivilegeKey  string            `json:"privilege_key"`
	RunId         string            `json:"run_id"`
	Metas         map[string]string `json:"metas"`
	ClientAddress string            `json:"client_address"`
}

type FrpLoginRequest struct {
	Content FrpLogin `json:"content"`
}

type FrpNewProxy struct {
	User  string            `json:"user"`
	Metas map[string]string `json:"metas"`

	ProxyName string `json:"proxy_name"`
	ProxyType string `json:"proxy_type"`

	RemotePort int `json:"remote_port"`

	CustomDomains []string `json:"custom_domains"`
	Subdomain     string   `json:"subdomain"`
}

type FrpNewProxyRequest struct {
	Content FrpNewProxy `json:"content"`
}

func handleReq(res http.ResponseWriter, req *http.Request) {
	// Read query parameters
	query := req.URL.Query()
	op := query.Get("op")

	// Read body
	decoder := json.NewDecoder(req.Body)

	// Handle operation
	switch op {
	case "":
		log.Printf("Missing 'op' in URL")
		http.Error(res, "Missing 'op' in URL", 400)
		return
	case "Login":
		var body FrpLoginRequest
		err := decoder.Decode(&body)
		if err != nil {
			log.Printf("Bad JSON in request: %s", err)
			http.Error(res, "Bad JSON", 400)
			return
		}
		// TODO
		log.Printf("Login: %s %s %s", body.Content.ClientAddress, body.Content.User, body.Content.Metas)
	case "NewProxy":
		var body FrpNewProxyRequest
		err := decoder.Decode(&body)
		if err != nil {
			log.Printf("Bad JSON in request: %s", err)
			http.Error(res, "Bad JSON", 400)
			return
		}
		// TODO
	default:
		res.WriteHeader(200)
		io.WriteString(res, "{\"reject\": false, \"unchange\": true}")
	}
}
