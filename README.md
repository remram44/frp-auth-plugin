# Auth server plugin for frp

This allows you to have multiple clients, each with a specific username and password, that can only create specific proxies. The configuration of the proxies is controlled by the server rather than the client.

Example `frp-auth.json`:

```json
{
    "users": [
        {
            "username": "frpc1",
            "password": "1234",
            "proxies": [
                {
                    "name": "frpcUser.service1",
                    "custom_domains": ["service1.127-0-0-1.nip.io", "service.127-0-0-1.nip.io"],
                    "http_user": "enduser",
                    "http_password": "hackme"
                }
            ]
        }
    ]
}
```

Run frp-auth-plugin like so:

```
FRP_AUTH_PLUGIN_LISTEN_ADDR=:8080 ./frp-auth-plugin frp-auth.json
```

Example `frps.yaml`:

```yaml
bindPort: 8000
vhostHTTPPort: 8000

httpPlugins:
  - name: "auth"
    addr: "127.0.0.1:8080" # Point to frp-auth-plugin
    path: "/handler"
    ops:
      - Login
      - NewProxy
```

Example `frpc.yaml`:

```yaml
serverAddr: 127.0.0.1
serverPort: 8001
transport:
  protocol: websocket
  heartbeatInterval: 10
  heartbeatTimeout: 30

# Match the credentials here
user: "frpc1"
metadatas:
  token: "1234"

proxies:
    # Match the service name here
  - name: service1
    type: http

    # Local destination
    localIp: 127.0.0.1
    localPort: 3000

    # You don't have to set this, and it will be overwritten if you do
    #customDomains:
    #  - service1.127-0-0-1.nip.io
    #httpUser: enduser 
    #httpPassword: hackme
```
