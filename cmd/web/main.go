package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sandromello/wgadmin/pkg/api"
	"github.com/sandromello/wgadmin/pkg/cli"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
)

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

func main() {
	http.HandleFunc("/peers/", configurePeerHandler)
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8000"
	}
	if err := cli.CreateConfigPath(nil, nil); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	log.Printf("Starting server at :%s", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		fmt.Println(err)
	}
}

func configurePeerHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	vpn := r.URL.Query().Get("vpn")
	client, err := storeclient.NewGCS(cli.DBFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	peerList, err := client.Peer().List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var peer *api.Peer
	for _, p := range peerList {
		if p.SecretValue == parts[1] {
			peer = &p
			break
		}
	}
	if peer == nil {
		http.Error(w, "Error: peer not found for this token.", http.StatusNotFound)
		return
	}
	if peer.Status == api.PeerStatusBlocked {
		http.Error(w, "Error: peer blocked, contact the administrator!", http.StatusBadRequest)
		return
	}
	updAt, err := time.Parse(time.RFC3339, peer.UpdatedAt)
	if err != nil {
		http.Error(w, "Error: failed parsing updated time for peer!", http.StatusInternalServerError)
		return
	}
	if updAt.Add(time.Minute * 30).Before(time.Now().UTC()) {
		msg := fmt.Sprintf("Error: secret has expired, updated at: %v!", peer.UpdatedAt)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	clientPrivkey, err := api.GeneratePrivateKey()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	wgsc, err := client.WireguardServerConfig().Get(vpn)
	if wgsc == nil || err != nil {
		msg := fmt.Sprintf("Error: failed retrieving wireguard server config object: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := api.HandleTemplates(string(templateWireguardClientConfig), &buf, map[string]interface{}{
		"PrivateKey": clientPrivkey,
		"PublicKey":  wgsc.PrivateKey.PublicKey(),
		"Address":    peer.AllowedIPs.String(),
		"DNS":        "1.1.1.1, 8.8.8.8",
		"Endpoint":   wgsc.PublicEndpoint,
		"AllowedIPs": "0.0.0.0/0, ::/0",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pubkey := clientPrivkey.PublicKey()
	peer.PublicKey = &pubkey
	peer.Status = api.PeerStatusActive
	// it's important to let the client to download the
	// configuration only once for security concerns.
	peer.SecretValue = ""
	if err := client.Peer().Update(peer); err != nil {
		msg := fmt.Sprintf("Error: failed updating peer: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	if err := client.SyncGCS(); err != nil {
		msg := fmt.Sprintf("Error: failed syncing with GCS: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	fmt.Fprintf(w, buf.String())
}
