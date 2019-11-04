package wgtools

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

// FileExists will return an error if the file exists
// or it can't read the file for some reason
func FileExists(filePath string) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil
	} else if err == nil {
		return fmt.Errorf("file path exists")
	}
	return err
}

// IsRootUser check if the current user is root
func IsRootUser() (bool, error) {
	u, err := user.Current()
	if err != nil {
		return false, err
	}
	return u.Username == "root", nil
}

// WGQuickUP turn on a wireguard interface
// https://git.zx2c4.com/WireGuard/about/src/tools/man/wg-quick.8
func WGQuickUP(configPath string) ([]byte, error) {
	cmd := exec.Command("wg-quick", "up", configPath)
	return cmd.CombinedOutput()
}

// WGQuickDown turn off a wireguard interface
// https://git.zx2c4.com/WireGuard/about/src/tools/man/wg-quick.8
func WGQuickDown(configPath string) ([]byte, error) {
	cmd := exec.Command("wg-quick", "down", configPath)
	return cmd.CombinedOutput()
}

// WGShowPeers list all local peers
func WGShowPeers(iface string) ([]string, error) {
	cmd := exec.Command("wg", "show", iface, "peers")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v. %v", string(output), err)
	}
	if len(output) == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimSuffix(string(output), "\n"), "\n"), nil
}

// WGAddPeer add a local peer into Wireguard
func WGAddPeer(iface, pubKey, allowedIPs string) ([]byte, error) {
	cmd := exec.Command("wg", "set", iface, "peer", pubKey, "allowed-ips", allowedIPs)
	return cmd.CombinedOutput()
}

// WGRemovePeer remove a local peer from Wireguard
func WGRemovePeer(iface, pubKey string) ([]byte, error) {
	cmd := exec.Command("wg", "set", iface, "peer", pubKey, "remove")
	return cmd.CombinedOutput()
}
