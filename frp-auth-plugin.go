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

var configFile *configfile.ConfigFile

func main() {
	ctx := context.Background()

	// Read command line
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: frp-auth-plugin <configuration-file>\n")
		os.Exit(2)
	}
	configFileName := os.Args[1]

	// Read configuration
	var err error
	configFile, err = configfile.New(configFileName, ctx)
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

type FrpNewProxyUser struct {
	User  string            `json:"user"`
	Metas map[string]string `json:"metas"`
}

type FrpNewProxy struct {
	User  FrpNewProxyUser   `json:"user"`
	Metas map[string]string `json:"metas"`

	ProxyName string `json:"proxy_name"`
	ProxyType string `json:"proxy_type"`

	RemotePort int `json:"remote_port"`

	CustomDomains []string `json:"custom_domains"`

	HttpUser     string `json:"http_user,omitempty"`
	HttpPassword string `json:"http_pwd,omitempty"`
}

type FrpNewProxyRequest struct {
	Content FrpNewProxy `json:"content"`
}

func handleReq(res http.ResponseWriter, req *http.Request) {
	// Read query parameters
	query := req.URL.Query()
	op := query.Get("op")

	reject := func(reason string) {
		res.WriteHeader(200)
		io.WriteString(res, "{\"reject\": true, \"reject_reason\": \"")
		io.WriteString(res, reason)
		io.WriteString(res, "\"}")
	}

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

		// Lookup user
		config := configFile.CurrentConfig()
		for _, user := range config.Users {
			if user.Username != body.Content.User {
				continue
			}

			if user.Password != body.Content.Metas["token"] {
				log.Printf("Invalid password for %s", body.Content.User)
				reject("invalid password")
				return
			}

			log.Printf("Login from %s as %s", body.Content.ClientAddress, body.Content.User)
			res.WriteHeader(200)
			io.WriteString(res, "{\"reject\": false, \"unchange\": true}")
			return
		}

		log.Printf("Invalid user %#v", body.Content.User)
		reject("invalid user")
	case "NewProxy":
		var body FrpNewProxyRequest
		err := decoder.Decode(&body)
		if err != nil {
			log.Printf("Bad JSON in request: %s", err)
			http.Error(res, "Bad JSON", 400)
			return
		}

		// Lookup user
		config := configFile.CurrentConfig()
		for _, user := range config.Users {
			if user.Username != body.Content.User.User {
				continue
			}

			// Lookup proxy
			for _, proxy := range user.Proxies {
				if proxy.Name != body.Content.ProxyName {
					continue
				}

				if body.Content.ProxyType != "http" {
					reject("proxy not http")
					return
				}

				// Replace proxy config with our own
				newProxy := FrpNewProxy{
					User:          body.Content.User,
					Metas:         body.Content.Metas,
					ProxyName:     body.Content.ProxyName,
					ProxyType:     body.Content.ProxyType,
					CustomDomains: proxy.CustomDomains,
					HttpUser:      proxy.HttpUser,
					HttpPassword:  proxy.HttpPassword,
				}

				log.Printf("NewProxy %s from %s", body.Content.ProxyName, body.Content.User.User)
				res.WriteHeader(200)
				io.WriteString(res, "{\"reject\": false, \"unchange\": false, \"content\":")
				encoder := json.NewEncoder(res)
				encoder.Encode(newProxy)
				io.WriteString(res, "}")
				return
			}

			log.Printf("Invalid proxy %s %s", body.Content.User, body.Content.ProxyName)
			return
		}

		log.Printf("Invalid user %#v", body.Content.User)
		reject("invalid user")
	default:
		res.WriteHeader(200)
		io.WriteString(res, "{\"reject\": false, \"unchange\": true}")
	}
}
