package api

var templateWireguardServerConfig = []byte(`[Interface]
Address    = {{ .Address }}
ListenPort = {{ .ListenPort }}
PrivateKey = {{ .PrivateKey }}

{{ range .PostUp -}}
PostUp = {{ . }}
{{ end }}
{{ range .PostDown -}}
PostDown = {{ . }}
{{ end }}`)

var templateWireguardClientConfig = []byte(`[Interface]
PrivateKey = {{ .PrivateKey.String }}
Address    = {{ .Address }}
DNS        = {{ .DNS }}
MTU        = 1360

[Peer]
PublicKey           = {{ .PublicKey }}
AllowedIPs          = {{ .AllowedIPs }}
Endpoint            = {{ .Endpoint }}
PersistentKeepalive = 25
`)
