package cli

import (
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/sandromello/wgadmin/web"
	"github.com/spf13/cobra"
)

// RunWebServerCmd start the webserver
// https://console.developers.google.com/apis/dashboard
func RunWebServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "run-server",
		Short:             "Run the client configuration generator webserver.",
		PersistentPreRunE: PersistentPreRunE,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			mux := http.NewServeMux()

			// Static Files
			staticDir := path.Join("", "web/static")
			fs := http.FileServer(http.Dir(staticDir))
			mux.Handle("/static/", http.StripPrefix("/static/", fs))

			handler := web.NewHandler(
				[]byte(`mykey`),
				&O.WebServer.PageConfig,
				*O.WebServer.AllowedDomains,
			)
			mux.HandleFunc("/", handler.Index)
			mux.HandleFunc("/signin", handler.Signin)
			mux.HandleFunc("/signout/", handler.Signout)
			mux.HandleFunc("/peers/", handler.Peers)
			address := fmt.Sprintf(":%s", O.WebServer.HTTPPort)
			log.Printf("Starting the webserver at :%s ...", address)
			if O.WebServer.TLSKeyFile != "" && O.WebServer.TLSCertFile != "" {
				return http.ListenAndServeTLS(
					address,
					O.WebServer.TLSCertFile,
					O.WebServer.TLSKeyFile,
					mux,
				)
			}
			return http.ListenAndServe(fmt.Sprintf(":%s", O.WebServer.HTTPPort), mux)
		},
	}
	pagec := &O.WebServer.PageConfig
	cmd.Flags().StringVar(&O.WebServer.HTTPPort, "port", "8000", "The port of the server.")
	cmd.Flags().StringVar(&O.WebServer.TLSKeyFile, "tls-key-file", "", "The certificate private key path.")
	cmd.Flags().StringVar(&O.WebServer.TLSCertFile, "tls-cert-file", "", "The certificate path.")
	O.WebServer.AllowedDomains = cmd.Flags().StringSlice("allowed-domains", []string{}, "A list of permitted domains that will be able to sign in.")
	cmd.Flags().StringVar(&pagec.GoogleClientID, "google-client-id", "", "The Google Client ID.")
	cmd.Flags().StringVar(&pagec.GoogleRedirectURI, "google-redirect-uri", "", "The Google Redirect URI address.")
	cmd.Flags().StringVar(&pagec.Title, "page-title", "VPN Service", "The title of the page.")
	cmd.Flags().StringVar(&pagec.TemplatePath, "template-path", "web/templates", "The path of html file templates.")
	cmd.Flags().StringVar(&pagec.ThemeCSSURL, "theme-css-url", "static/themes/default/styles.css", "The CSS theme.")
	cmd.Flags().StringVar(&pagec.LogoURL, "logo-url", "static/img/logo.png", "The logo URL.")
	cmd.Flags().StringVar(&pagec.FaviconURL, "favicon-url", "", "The favicon URL.")
	cmd.Flags().StringVar(&pagec.NavBarLink, "navbar-link", "https://github.com/sandromello/wgadmin", "Custom link for the navbar logo.")
	return cmd
}
