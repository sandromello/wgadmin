package api

var templateWireguardServerConfig = []byte(`[Interface]
Address    = {{ .Address.String }}
ListenPort = {{ .ListenPort }}
PrivateKey = {{ .PrivateKey.String }}

{{ range .PostUp -}}
PostUp = {{ . }}
{{ end }}
{{ range .PostDown -}}
PostDown = {{ . }}
{{ end }}`)

var templateWireguardServerPeersConfig = []byte(`
{{- range .Peers -}}
[Peer]
PublicKey = {{ .PublicKey.String }}
AllowedIPs = {{ .AllowedIPs }}

{{ end }}`)

var templateWireguardClientConfig = []byte(`[Interface]
PrivateKey = {{ .InterfaceClientConfig.PrivateKey.String }}
Address    = {{ .InterfaceClientConfig.Address }}
DNS        = {{ .InterfaceClientConfig.ParseDNSToComma }}

[Peer]
PublicKey  = {{ .PeerClientConfig.PublicKey }}
AllowedIPs = {{ .PeerClientConfig.ParseAllowedIPsToComma }}
Endpoint   = {{ .PeerClientConfig.Endpoint }}

PersistentKeepalive = {{ .PeerClientConfig.PersistentKeepAlive }}
`)
