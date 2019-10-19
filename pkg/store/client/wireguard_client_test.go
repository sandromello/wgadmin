package client

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sandromello/wgadmin/pkg/api"
	bolt "go.etcd.io/bbolt"
)

func openTempFile(path string, flag int, mode os.FileMode) (*os.File, error) {
	f, err := ioutil.TempFile("", "vpnapp-")
	if err != nil {
		return nil, err
	}
	return f, os.Remove(f.Name())
}

func encode(obj interface{}) []byte {
	encoded, _ := json.Marshal(obj)
	return encoded
}

func TestWireguardClientStoreSetGet(t *testing.T) {
	var resourceUID = "test@domain.tld"
	c := NewOrDie("", "wg-prod", &bolt.Options{OpenFile: openTempFile})
	privkey, err := api.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("failed generating private key: %v", err)
	}

	wgcc := &api.WireguardClientConfig{
		UID: resourceUID,
		InterfaceClientConfig: api.InterfaceClientConfig{
			PrivateKey: &privkey,
			Address:    api.ParseCIDR("192.168.0.100/32"),
			DNS:        []net.IP{net.ParseIP("1.1.1.1"), net.ParseIP("8.8.8.8")},
		},
		PeerClientConfig: api.PeerClientConfig{
			PublicKey:           privkey.PublicKey().String(),
			AllowedIPs:          api.ParseAllowedIPs("0.0.0.0/0", "::/0"),
			Endpoint:            "wg-dev.vpn.domain.tld:51820",
			PersistentKeepAlive: 25,
		},
	}
	if err := c.WireguardClientConfig().Create(wgcc); err != nil {
		t.Fatalf("failed creating wireguard client config: %v", err)
	}
	wgccExpected, err := c.WireguardClientConfig().Get(resourceUID)
	if err != nil {
		t.Fatalf("failed retrieving expected wireguard client config: %v", err)
	}
	if diff := cmp.Diff(encode(wgcc), encode(wgccExpected)); diff != "" {
		t.Fatalf("unexpected object (-want +got):\n%s", diff)
	}
}

func TestWireguardClientStoreListAndDelete(t *testing.T) {
	clients := []api.WireguardClientConfig{
		{
			UID:                   "alpha@domain.tld",
			InterfaceClientConfig: api.InterfaceClientConfig{Address: api.ParseCIDR("10.100.0.10/32")},
			PeerClientConfig:      api.PeerClientConfig{Endpoint: "wg-dev.vpn.domain.tld:51820"},
		},
		{
			UID:                   "beta@domain.tld",
			InterfaceClientConfig: api.InterfaceClientConfig{Address: api.ParseCIDR("10.100.0.11/32")},
			PeerClientConfig:      api.PeerClientConfig{Endpoint: "wg-dev.vpn.domain.tld:51820"},
		},
		{
			UID:                   "gama@domain.tld",
			InterfaceClientConfig: api.InterfaceClientConfig{Address: api.ParseCIDR("10.100.0.12/32")},
			PeerClientConfig:      api.PeerClientConfig{Endpoint: "wg-dev.vpn.domain.tld:51820"},
		},
	}
	c := NewOrDie("", "wg-prod", &bolt.Options{OpenFile: openTempFile})
	var expectedList []string
	for _, obj := range clients {
		expectedList = append(expectedList, string(encode(obj)))
		if err := c.WireguardClientConfig().Create(&obj); err != nil {
			t.Fatalf("failed creating wg client config: %v", err)
		}
	}

	wgccList, _ := c.WireguardClientConfig().List()
	var resultList []string
	for _, wgcc := range wgccList {
		resultList = append(resultList, string(encode(wgcc)))
		if err := c.WireguardClientConfig().Delete(wgcc.UID); err != nil {
			t.Fatalf("failed removing obj UID %v: %v", wgcc.UID, err)
		}
	}
	// Test List()
	if diff := cmp.Diff(resultList, expectedList); diff != "" {
		t.Fatalf("unexpected result (-want +got):\n%s", diff)
	}
	// Test Delete()
	wgccList, _ = c.WireguardClientConfig().List()
	var expectedDeletedList []api.WireguardClientConfig
	if diff := cmp.Diff(wgccList, expectedDeletedList); diff != "" {
		t.Fatalf("unexpected result (-want +got):\n%s", diff)
	}
}
