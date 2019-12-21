package wgserver

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/go-systemd/dbus"
)

const (
	systemdPath = "/etc/systemd/system"
	unitName    = "wgadmin-server.service"
)

var wgAdminServerUnitTemplate = []byte(
	`[Unit]
Description=wgadmin-server: It manages wireguard servers
[Service]
Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
EnvironmentFile=-/etc/default/wgadmin
ExecStart=wgadmin configure server $WGADMIN_SERVER --config $WGADMIN_CONFIG_FILE --sync $WGADMIN_SERVER_SYNC
Restart=always
StartLimitInterval=0
RestartSec=300
[Install]
WantedBy=multi-user.target
`)

// StopWireguardManager stop the systemd unit
func StopWireguardManager() error {
	conn, err := dbus.New()
	if err != nil {
		return fmt.Errorf("failed getting dbus connection, err=%v", err)
	}
	defer conn.Close()
	response := make(chan string)
	_, err = conn.StopUnit(unitName, "replace", response)
	if err != nil {
		return err
	}
	select {
	case status := <-response:
		return fmt.Errorf(status)
	case <-time.After(120 * time.Second):
		return fmt.Errorf("timeout(2m) on stopping %q", unitName)
	}
}

// InstallWireguardManager copy the template files to systemd enable and start the service
func InstallWireguardManager() error {
	// Install Kubelet DropIn
	if err := os.MkdirAll("/etc/wireguard", 0755); err != nil {
		return fmt.Errorf("failed wireguard folder: %v", err)
	}
	wgadminSystemdPath := filepath.Join(systemdPath, unitName)
	// var buf bytes.Buffer
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
	if _, err := conn.StartUnit(unitName, "replace", nil); err != nil {
		return fmt.Errorf("failed starting unit, err=%v", err)
	}
	return nil
}
