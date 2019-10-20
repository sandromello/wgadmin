package cli

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sandromello/wgadmin/pkg/api"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/spf13/cobra"
)

func configurePeerHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	vpn := r.URL.Query().Get("vpn")
	client, err := storeclient.New(GlobalDBFile, GlobalBoltOptions)
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
	if wgsc == nil && err == nil {
		msg := fmt.Sprintf("Error: the wireguard server %q doesn't exists", vpn)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if err != nil {
		msg := fmt.Sprintf("Error: failed retrieving wireguard server config object: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	data, err := api.ParseWireguardClientConfigTemplate(map[string]interface{}{
		"PrivateKey": clientPrivkey,
		"PublicKey":  wgsc.PrivateKey.PublicKey(),
		"Address":    peer.AllowedIPs.String(),
		"DNS":        "1.1.1.1, 8.8.8.8",
		"Endpoint":   wgsc.PublicEndpoint,
		"AllowedIPs": "0.0.0.0/0, ::/0",
	})
	if err != nil {
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
	if err := client.SyncRemote(); err != nil {
		msg := fmt.Sprintf("Error: failed syncing with GCS: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	fmt.Fprintf(w, string(data))
}

// RunWebServerCmd start the webserver
func RunWebServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run-server",
		Short:             "Run the client configuration generator webserver.",
		PersistentPreRunE: PersistentPreRunE,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			http.HandleFunc("/peers/", configurePeerHandler)
			log.Printf("Starting the webserver at :%s ...", O.WebServer.HTTPPort)
			return http.ListenAndServe(fmt.Sprintf(":%s", O.WebServer.HTTPPort), nil)
		},
	}
	cmd.Flags().StringVar(&O.WebServer.HTTPPort, "port", "8000", "The port of the server.")
	return cmd
}
