package wgtools

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
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
