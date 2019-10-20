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
PrivateKey = {{ .PrivateKey.String }}
Address    = {{ .Address }}
DNS        = {{ .DNS }}
MTU        = 1360

[Peer]
PublicKey           = {{ .PublicKey.String }}
AllowedIPs          = {{ .AllowedIPs }}
Endpoint            = {{ .Endpoint }}
PersistentKeepalive = 25
`)
