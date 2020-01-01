package client

import (
	"encoding/json"
	"io/ioutil"
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

func TestWireguardServerListAndDelete(t *testing.T) {
	c := NewOrDie("", "", &bolt.Options{OpenFile: openTempFile})
	privkey01, _ := api.GeneratePrivateKey()
	privkey02, _ := api.GeneratePrivateKey()
	privkey03, _ := api.GeneratePrivateKey()
	// cipherKey, _ := util.NewAESCipherKey("")

	var expectedJSONList []string
	for _, w := range []api.WireguardServerConfig{
		{
			Metadata:            api.Metadata{UID: "dev"},
			Address:             "10.100.0.10/32",
			ListenPort:          51820,
			EncryptedPrivateKey: privkey01.String(),
			PublicKey:           func() *api.Key { k := privkey01.PublicKey(); return &k }(),
			PostUp:              []string{"ip link set mtu 1500 dev ens4"},
			PostDown:            []string{"sysctl -w net.ipv4.ip_forward=0"},
		},
		{
			Metadata:            api.Metadata{UID: "prod"},
			Address:             "10.100.0.11/32",
			ListenPort:          51820,
			EncryptedPrivateKey: privkey02.String(),
			PublicKey:           func() *api.Key { k := privkey02.PublicKey(); return &k }(),
			PostUp:              []string{"ip link set mtu 1500 dev ens4"},
			PostDown:            []string{"sysctl -w net.ipv4.ip_forward=0"},
		},
		{
			Metadata:            api.Metadata{UID: "staging"},
			Address:             "10.100.0.12/32",
			ListenPort:          51820,
			EncryptedPrivateKey: privkey03.String(),
			PublicKey:           func() *api.Key { k := privkey03.PublicKey(); return &k }(),
			PostUp:              []string{"ip link set mtu 1500 dev ens4"},
			PostDown:            []string{"sysctl -w net.ipv4.ip_forward=0"},
		},
	} {
		if err := c.WireguardServerConfig().Update(&w); err != nil {
			t.Fatalf("failed updating wireguard server config: %v", err)
		}
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatalf("failed serializing expected wireguard config %v: %v", w.UID, err)
		}
		// the uid's are sorted when listing servers,
		// so the uid's of the expected objects must be sorted
		expectedJSONList = append(expectedJSONList, string(data))
	}
	wgscList, err := c.WireguardServerConfig().List()
	if err != nil {
		t.Fatalf("failed listing wireguard server config: %v", err)
	}
	var gotJSONList []string
	for _, w := range wgscList {
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatalf("failed serializing wanted wireguard server config %v: %v", w.UID, err)
		}
		gotJSONList = append(gotJSONList, string(data))
	}
	if diff := cmp.Diff(expectedJSONList, gotJSONList); diff != "" {
		t.Fatalf("unexpected object (-want +got):\n%s", diff)
	}
	for _, w := range wgscList {
		if err := c.WireguardServerConfig().Delete(w.UID); err != nil {
			t.Fatalf("failed deleting wireguard server config: %v", err)
		}
	}
	var expectedDeletedList []api.WireguardServerConfig
	wgscList, _ = c.WireguardServerConfig().List()
	if diff := cmp.Diff(wgscList, expectedDeletedList); diff != "" {
		t.Fatalf("unexpected object (-want +got):\n%s", diff)
	}
}

func TestWireguardServerStoreCRUD(t *testing.T) {
	var bucketName = "wg-prod"
	c := NewOrDie("", bucketName, &bolt.Options{OpenFile: openTempFile})
	privkey, err := api.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("failed generating private key: %v", err)
	}

	wgsc := &api.WireguardServerConfig{
		Metadata:            api.Metadata{UID: bucketName},
		Address:             "10.100.0.10/32",
		ListenPort:          51820,
		EncryptedPrivateKey: privkey.String(),
		PublicKey:           func() *api.Key { k := privkey.PublicKey(); return &k }(),
		PostUp: []string{
			"ip link set mtu 1500 dev ens4",
			"ip link set mtu 1500 dev %i",
			"sysctl -w net.ipv4.ip_forward=1",
		},
		PostDown: []string{
			"sysctl -w net.ipv4.ip_forward=0",
			"iptables -D FORWARD -o %i -j ACCEPT",
			"iptables -D FORWARD -i %i -j ACCEPT",
		},
	}
	if err := c.WireguardServerConfig().Update(wgsc); err != nil {
		t.Fatalf("failed creating wg server config: %v", err)
	}
	wgscExpected, err := c.WireguardServerConfig().Get(bucketName)
	if err != nil {
		t.Fatalf("failed retrieving expected wg server config: %v", err)
	}
	if diff := cmp.Diff(encode(wgsc), encode(wgscExpected)); diff != "" {
		t.Fatalf("unexpected object (-want +got):\n%s", diff)
	}
}
