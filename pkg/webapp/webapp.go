package webapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/sessions"
	"github.com/sandromello/wgadmin/pkg/api"
	"github.com/sandromello/wgadmin/pkg/store"
	storeclient "github.com/sandromello/wgadmin/pkg/store/client"
	"github.com/sandromello/wgadmin/pkg/util"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const (
	indexPageName          = "index.html"
	loginPageName          = "login.html"
	errorPageName          = "error.html"
	sessionMaxAgeInSeconds = 3600
)

// PageConfig is used to configure the content of the webapp
type PageConfig struct {
	FaviconURL        string
	LogoURL           string
	ThemeCSSURL       string
	GoogleClientID    string
	GoogleRedirectURI string
	TemplatePath      string
	Title             string
	NavBarLink        string
}

// UserInfo represents an Google user
type UserInfo struct {
	jwt.StandardClaims

	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	GSuiteDomain  string `json:"hd"`
	Locale        string `json:"locale"`
	Picture       string `json:"picture"`
}

// ToJSON converts a *UserInfo to json
func (u *UserInfo) ToJSON() []byte {
	data, err := json.Marshal(u)
	if err != nil {
		return nil
	}
	return data
}

// UnmarshalUserInfo unmarshal to *UserInfo type
func UnmarshalUserInfo(data []byte) *UserInfo {
	u := &UserInfo{}
	if err := json.Unmarshal(data, u); err != nil {
		return nil
	}
	return u
}

// Handler containing configuration for handlers
type Handler struct {
	tmpl           *template.Template
	store          *sessions.CookieStore
	pageConfig     *PageConfig
	allowedDomains []string
}

// NewHandler creates a new handler
func NewHandler(sessionKey []byte, pconfig *PageConfig, allowedDomains []string) *Handler {
	if pconfig == nil {
		log.Fatal("page config attribute is nil")
	}
	h := &Handler{
		store:          sessions.NewCookieStore(sessionKey),
		pageConfig:     pconfig,
		allowedDomains: allowedDomains,
	}

	h.RenderTemplates()
	h.store.MaxAge(sessionMaxAgeInSeconds)
	return h
}

func (h *Handler) isAllowedDomain(email string) (bool, string) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false, ""
	}
	for _, d := range h.allowedDomains {
		if d == parts[1] {
			return true, d
		}
	}
	return false, parts[1]
}

func (h *Handler) getSessionUser(r *http.Request) (*UserInfo, error) {
	session, err := h.store.Get(r, "wgadmin")
	if err != nil {
		return nil, err
	}
	if data, ok := session.Values["userinfo"]; ok {
		return UnmarshalUserInfo(data.([]byte)), nil
	}
	return nil, nil
}

// RenderTemplates load all templates
func (h *Handler) RenderTemplates() {
	indexPage := filepath.Join("", h.pageConfig.TemplatePath, indexPageName)
	loginPage := filepath.Join("", h.pageConfig.TemplatePath, loginPageName)
	errorPage := filepath.Join("", h.pageConfig.TemplatePath, errorPageName)
	tmpl, err := template.New("").ParseFiles(indexPage, loginPage, errorPage)
	if err != nil {
		log.Fatalf("failed rendering templates: %v", err)
	}
	h.tmpl = tmpl
}

// Index the main page
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ENV") != "production" {
		h.RenderTemplates()
	}
	switch r.Method {
	case "POST":
		session, err := h.store.Get(r, "wgadmin")
		if err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if data, ok := session.Values["userinfo"]; ok {
			u := UnmarshalUserInfo(data.([]byte))
			if u != nil {
				expireAt := time.Unix(u.ExpiresAt, 0).Sub(time.Now().UTC())
				log.Infof("user %s already logged in, expires in %v minute(s). Redirecting ...", u.Email, int(expireAt.Minutes()))
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		idToken := r.FormValue("id_token")
		token, err := jwt.ParseWithClaims(idToken, &UserInfo{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(``), nil
		})
		if u, ok := token.Claims.(*UserInfo); ok {
			if ok, d := h.isAllowedDomain(u.Email); !ok {
				msg := fmt.Sprintf("Users from domain %s aren't allowed to signin!", d)
				h.httpError(w, msg, http.StatusUnauthorized)
				return
			}
			if !u.EmailVerified {
				h.httpError(w, "Email not verified", http.StatusUnauthorized)
				return
			}
			session.Values["userinfo"] = u.ToJSON()
			expireAt := time.Unix(u.ExpiresAt, 0).Sub(time.Now().UTC())
			log.Infof("user %v signed in, expires in %v minutes", u.Email, int(expireAt.Minutes()))
			session.Options.MaxAge = int(expireAt.Seconds())
			if err := session.Save(r, w); err != nil {
				h.httpError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
		}
	case "GET":
		u, err := h.getSessionUser(r)
		if err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if u == nil {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}
		// TODO: refactor
		configPath := filepath.Join(os.Getenv("$HOME/.wgapp/"), store.DBFileName)
		client, err := storeclient.New(configPath, &bolt.Options{OpenFile: storeclient.FetchFromGCS})
		if err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer client.Close()
		peerList, err := client.Peer().List()
		if err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var peerUserList []api.Peer
		for _, p := range peerList {
			parts := strings.Split(p.UID, "/")
			if len(parts) != 2 || u.Email != parts[1] {
				continue
			}
			peerUserList = append(peerUserList, p)
		}
		if err := h.tmpl.ExecuteTemplate(w, indexPageName, map[string]interface{}{
			"User":       u,
			"Peers":      peerUserList,
			"PageConfig": h.pageConfig,
		}); err != nil {
			log.Errorf("failed executing template: %v", err)
		}
	default:
		h.httpError(w, "Method Not Implemented", http.StatusNotImplemented)
	}
}

// Signin the webapp login page
func (h *Handler) Signin(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ENV") != "production" {
		h.RenderTemplates()
	}
	if err := h.tmpl.ExecuteTemplate(w, loginPageName, map[string]interface{}{
		"PageConfig": h.pageConfig,
	}); err != nil {
		log.Errorf("failed executing template: %v", err)
	}
}

// Signout removes data from session
func (h *Handler) Signout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.httpError(w, "Method Not Implemented", http.StatusNotImplemented)
		return
	}
	session, err := h.store.Get(r, "wgadmin")
	if err != nil {
		h.httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Delete Session
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		h.httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

// Peers generates and retrieve wireguard client configurations,
// only authenticated are allowed to download.
func (h *Handler) Peers(w http.ResponseWriter, r *http.Request) {
	u, err := h.getSessionUser(r)
	if err != nil {
		h.httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	configPath := filepath.Join(os.Getenv("$HOME/.wgapp/"), store.DBFileName)
	client, err := storeclient.New(configPath, &bolt.Options{OpenFile: storeclient.FetchFromGCS})
	if err != nil {
		h.httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	switch r.Method {
	case "POST":
		// expect <server>/<peer>
		peerUID := r.FormValue("peer_uid")
		peer, err := client.Peer().Get(peerUID)
		if err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if peer == nil {
			h.httpError(w, "Peer not found!", http.StatusNotFound)
			return
		}
		if !u.EmailVerified {
			h.httpError(w, "E-mail not verified!", http.StatusUnauthorized)
			return
		}
		log.Infof("got peer=%v, found=%v", peerUID, peer.UID)
		parts := strings.Split(peer.UID, "/")
		if len(parts) == 2 && parts[1] != u.Email {
			h.httpError(w, "Peer doesn't match with email", http.StatusUnauthorized)
			return
		}
		if peer.Status == api.PeerStatusBlocked {
			msg := fmt.Sprintf("Peer %s is blocked!", peer.UID)
			h.httpError(w, msg, http.StatusInternalServerError)
			return
		}
		// Reset Peer
		randomString, err := util.GenerateRandomString(50)
		if err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		peer.PublicKey = nil
		peer.Status = api.PeerStatusInitial
		peer.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		peer.SecretValue = fmt.Sprintf("%s.conf", randomString)
		if err := client.Peer().Update(peer); err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := client.SyncRemote(); err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		redirectURL := fmt.Sprintf("/peers/%s?vpn=%s", peer.SecretValue, peer.GetServer())
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	case "GET":
		peerList, err := client.Peer().List()
		if err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var peer *api.Peer
		secretParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if secretParts[1] == "" {
			h.httpError(w, "Not Found", http.StatusNotFound)
			return
		}
		for _, p := range peerList {
			parts := strings.Split(p.UID, "/")
			if len(parts) != 2 || u.Email != parts[1] {
				continue
			}
			if p.SecretValue != "" && p.SecretValue == secretParts[1] {
				peer = &p
				break
			}
		}
		if peer == nil {
			h.httpError(w, "Error: peer not found for this token.", http.StatusNotFound)
			return
		}
		if peer.Status == api.PeerStatusBlocked {
			h.httpError(w, "Error: peer blocked, contact the administrator!", http.StatusBadRequest)
			return
		}
		updAt, err := time.Parse(time.RFC3339, peer.UpdatedAt)
		if err != nil {
			h.httpError(w, "Error: failed parsing updated time for peer!", http.StatusInternalServerError)
			return
		}
		if updAt.Add(time.Minute * 15).Before(time.Now().UTC()) {
			msg := fmt.Sprintf("Error: secret has expired, updated at: %v!", peer.UpdatedAt)
			h.httpError(w, msg, http.StatusBadRequest)
			return
		}

		clientPrivkey, err := api.GeneratePrivateKey()
		if err != nil {
			h.httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vpn := r.URL.Query().Get("vpn")
		wgsc, err := client.WireguardServerConfig().Get(vpn)
		if wgsc == nil && err == nil {
			msg := fmt.Sprintf("Error: the wireguard server %q doesn't exists", vpn)
			h.httpError(w, msg, http.StatusBadRequest)
			return
		}
		if err != nil {
			msg := fmt.Sprintf("Error: failed retrieving wireguard server config object: %v", err)
			h.httpError(w, msg, http.StatusInternalServerError)
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
			h.httpError(w, err.Error(), http.StatusInternalServerError)
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
			h.httpError(w, msg, http.StatusInternalServerError)
			return
		}
		if err := client.SyncRemote(); err != nil {
			msg := fmt.Sprintf("Error: failed syncing with GCS: %v", err)
			h.httpError(w, msg, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		contentDisposition := fmt.Sprintf("attachment; filename=%s-%v.conf", vpn, time.Now().UTC().Unix())
		w.Header().Set("Content-Disposition", contentDisposition)
		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		io.Copy(w, bytes.NewBuffer(data))
	default:
		h.httpError(w, "Not Implemented", http.StatusNotImplemented)
	}
}

func (h *Handler) httpError(w http.ResponseWriter, msg string, code int) {
	if os.Getenv("ENV") != "production" {
		h.RenderTemplates()
	}
	w.WriteHeader(code)
	if err := h.tmpl.ExecuteTemplate(w, errorPageName, map[string]interface{}{
		"PageConfig": h.pageConfig,
		"StatusCode": code,
		"Message":    msg,
	}); err != nil {
		log.Errorf("failed executing template: %v", err)
	}
}
