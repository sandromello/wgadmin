package wgpeer

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/coreos/go-systemd/dbus"
)

const (
	systemdPath = "/etc/systemd/system"
	unitName    = "wgadmin-peer.service"
)

var wgAdminPeerUnitTemplate = []byte(
	`[Unit]
	Description=wgadmin-peer: It manages wireguard peers
	[Service]
	Environment="PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	EnvironmentFile=-/etc/default/wgadmin
	ExecStart=/usr/bin/wgadmin configure peer $WGADMIN_SERVER --sync $WGADMIN_PEER_SYNC
	Restart=always
	StartLimitInterval=0
	RestartSec=300
	[Install]
	WantedBy=multi-user.target
`)

// StopPeerManager stop the systemd unit
func StopPeerManager() error {
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

// InstallPeerManager copy the template files to systemd enable and start the service
func InstallPeerManager() error {
	wgadminSystemdPath := filepath.Join(systemdPath, unitName)
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
	if _, err := conn.StartUnit(unitName, "replace", nil); err != nil {
		return fmt.Errorf("failed starting unit, err=%v", err)
	}
	return nil
}
