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

			handler := web.NewHandler([]byte(`mykey`), &O.WebServer.PageConfig)
			mux.HandleFunc("/", handler.Index)
			mux.HandleFunc("/signin", handler.Signin)
			mux.HandleFunc("/signout/", handler.Signout)
			mux.HandleFunc("/peers/", handler.Peers)
			log.Printf("Starting the webserver at :%s ...", O.WebServer.HTTPPort)
			return http.ListenAndServe(fmt.Sprintf(":%s", O.WebServer.HTTPPort), mux)
		},
	}
	pagec := &O.WebServer.PageConfig
	cmd.Flags().StringVar(&O.WebServer.HTTPPort, "port", "8000", "The port of the server.")
	cmd.Flags().StringVar(&pagec.GoogleClientID, "google-client-id", "", "The Google Client ID.")
	cmd.Flags().StringVar(&pagec.GoogleRedirectURI, "google-redirect-uri", "", "The Google Redirect URI address.")
	cmd.Flags().StringVar(&pagec.Title, "page-title", "VPN Service", "The title of the page.")
	cmd.Flags().StringVar(&pagec.TemplatePath, "template-path", "web/templates", "The path of html file templates.")
	cmd.Flags().StringVar(&pagec.ThemeCSSURL, "theme-css-url", "static/themes/default/styles.css", "The CSS theme.")
	cmd.Flags().StringVar(&pagec.LogoURL, "logo-url", "static/img/logo.png", "The logo URL.")
	cmd.Flags().StringVar(&pagec.FaviconURL, "favicon-url", "", "The favicon URL.")
	return cmd
}
