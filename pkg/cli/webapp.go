package cli

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"

	"github.com/ghodss/yaml"
	"github.com/gorilla/securecookie"
	"github.com/sandromello/wgadmin/pkg/api"
	"github.com/sandromello/wgadmin/pkg/webapp"
	"github.com/spf13/cobra"
)

func parseWebAppConfigFile(path string) (*api.WebApp, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c api.WebApp
	return &c, yaml.Unmarshal(data, &c)
}

// RunWebServerCmd start the webserver
// https://console.developers.google.com/apis/dashboard
func RunWebServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run-server",
		Short:             "Run the client configuration generator webserver.",
		PersistentPreRunE: PersistentPreRunE,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			webappc, err := parseWebAppConfigFile(O.ServerConfigPath)
			if err != nil {
				return fmt.Errorf("failed parsing webapp config file: %v", err)
			}
			webappc.SetDefaults()
			mux := http.NewServeMux()

			// Static Files
			staticDir := path.Join("", "web/static")
			fs := http.FileServer(http.Dir(staticDir))
			mux.Handle("/static/", http.StripPrefix("/static/", fs))

			sessionKey := securecookie.GenerateRandomKey(32)
			if sessionKey == nil {
				return fmt.Errorf("failed generating session key")
			}
			handler := webapp.NewHandler(sessionKey, webappc.PageConfig, webappc.AllowedDomains)
			mux.HandleFunc("/", handler.Index)
			mux.HandleFunc("/signin", handler.Signin)
			mux.HandleFunc("/signout/", handler.Signout)
			mux.HandleFunc("/peers/", handler.Peers)
			address := fmt.Sprintf(":%s", webappc.HTTPPort)
			log.Printf("Starting the webserver at :%s ...", address)
			if webappc.TLSKeyFile != "" && webappc.TLSCertFile != "" {
				return http.ListenAndServeTLS(
					address,
					webappc.TLSCertFile,
					webappc.TLSKeyFile,
					mux,
				)
			}
			return http.ListenAndServe(fmt.Sprintf(":%s", webappc.HTTPPort), mux)
		},
	}
	cmd.Flags().StringVarP(&O.ServerConfigPath, "config-file", "c", "", "The wgadmin webapp config file.")
	return cmd
}
