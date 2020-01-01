package systemd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/go-systemd/dbus"
	"github.com/sandromello/wgadmin/pkg/api"
)

var (
	wgAdminServerUnitTemplate = []byte(
		`[Unit]
Description=wgadmin-server: It manages wireguard servers
[Service]
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
EnvironmentFile=-/etc/default/wgadmin
ExecStart=/usr/bin/wgadmin sync-servers --config-file $WGADMIN_CONFIG_PATH
Restart=always
StartLimitInterval=0
RestartSec=300
[Install]
WantedBy=multi-user.target
`)
	wgAdminPeerUnitTemplate = []byte(`[Unit]
Description=wgadmin-peer: It manages wireguard peers
[Service]
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
EnvironmentFile=-/etc/default/wgadmin
ExecStart=/usr/bin/wgadmin sync-peers --config-file $WGADMIN_CONFIG_PATH
Restart=always
StartLimitInterval=0
RestartSec=300
[Install]
WantedBy=multi-user.target
`)
)

// StopDaemon stop the systemd unit
func StopDaemon(d api.Daemon) error {
	conn, err := dbus.New()
	if err != nil {
		return fmt.Errorf("failed getting dbus connection, err=%v", err)
	}
	defer conn.Close()
	response := make(chan string)
	_, err = conn.StopUnit(d.GetUnitName(), "replace", response)
	if err != nil {
		return err
	}
	select {
	case status := <-response:
		return fmt.Errorf(status)
	case <-time.After(120 * time.Second):
		return fmt.Errorf("timeout(2m) on stopping %q", d.GetUnitName())
	}
}

// InstallWireguardManager copy the template files to systemd enable and start the service
func InstallWireguardManager(d *api.ServerDaemon) error {
	if err := os.MkdirAll(d.ConfigPath, 0755); err != nil {
		return fmt.Errorf("failed creating wireguard folder: %v", err)
	}
	wgadminSystemdPath := filepath.Join(d.SystemdPath, d.UnitName)
	if err := ioutil.WriteFile(wgadminSystemdPath, wgAdminServerUnitTemplate, 0644); err != nil {
		return fmt.Errorf("failed writing systemd unit. err=%v", err)
	}
	conn, err := dbus.New()
	if err != nil {
		return fmt.Errorf("failed getting dbus connection, err=%v", err)
	}
	defer conn.Close()
	_, changes, err := conn.EnableUnitFiles([]string{wgadminSystemdPath}, false, true)
	if err != nil {
		return fmt.Errorf("failed enabling unit, err=%v", err)
	}
	for _, chg := range changes {
		log.Printf("-> Created %s %s -> %s.", chg.Type, chg.Filename, chg.Destination)
	}
	if err := conn.Reload(); err != nil {
		log.Printf("failed reloading units, err=%v", err)
	}
	if _, err := conn.StartUnit(d.UnitName, "replace", nil); err != nil {
		return fmt.Errorf("failed starting unit, err=%v", err)
	}
	return nil
}

// InstallPeerManager copy the template files to systemd enable and start the service
func InstallPeerManager(d *api.PeerDaemon) error {
	wgadminSystemdPath := filepath.Join(d.GetSystemdPath(), d.GetUnitName())
	if err := ioutil.WriteFile(wgadminSystemdPath, wgAdminPeerUnitTemplate, 0644); err != nil {
		return fmt.Errorf("failed writing systemd unit. err=%v", err)
	}
	conn, err := dbus.New()
	if err != nil {
		return fmt.Errorf("failed getting dbus connection, err=%v", err)
	}
	defer conn.Close()
	_, changes, err := conn.EnableUnitFiles([]string{wgadminSystemdPath}, false, true)
	if err != nil {
		return fmt.Errorf("failed enabling unit, err=%v", err)
	}
	for _, chg := range changes {
		log.Printf("-> Created %s %s -> %s.", chg.Type, chg.Filename, chg.Destination)
	}
	if err := conn.Reload(); err != nil {
		log.Printf("failed reloading units, err=%v", err)
	}
	if _, err := conn.StartUnit(d.GetUnitName(), "replace", nil); err != nil {
		return fmt.Errorf("failed starting unit, err=%v", err)
	}
	return nil
}
