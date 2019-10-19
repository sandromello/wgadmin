package client

import (
	"encoding/json"
	"testing"

	// https://github.com/golang/go/issues/12153#issuecomment-229998750
	_ "golang.org/x/crypto/ripemd160"

	"github.com/google/go-cmp/cmp"
	"github.com/sandromello/wgadmin/pkg/api"
	bolt "go.etcd.io/bbolt"
)

func TestWireguardServerListAndDelete(t *testing.T) {
	c := NewOrDie("", "", &bolt.Options{OpenFile: openTempFile})
	privkey01, _ := api.GeneratePrivateKey()
	privkey02, _ := api.GeneratePrivateKey()
	privkey03, _ := api.GeneratePrivateKey()
	var expectedJSONList []string
	for _, w := range []api.WireguardServerConfig{
		{
			UID:        "dev",
			Address:    api.ParseCIDR("10.100.0.10/32"),
			ListenPort: 51820,
			PrivateKey: &privkey01,
			PostUp:     []string{"ip link set mtu 1500 dev ens4"},
			PostDown:   []string{"sysctl -w net.ipv4.ip_forward=0"},
		},
		{
			UID:        "prod",
			Address:    api.ParseCIDR("10.100.0.11/32"),
			ListenPort: 51820,
			PrivateKey: &privkey02,
			PostUp:     []string{"ip link set mtu 1500 dev ens4"},
			PostDown:   []string{"sysctl -w net.ipv4.ip_forward=0"},
		},
		{
			UID:        "staging",
			Address:    api.ParseCIDR("10.100.0.12/32"),
			ListenPort: 51820,
			PrivateKey: &privkey03,
			PostUp:     []string{"ip link set mtu 1500 dev ens4"},
			PostDown:   []string{"sysctl -w net.ipv4.ip_forward=0"},
		},
	} {
		if err := c.WireguardServerConfig().Create(&w); err != nil {
			t.Fatalf("failed creating wireguard server config: %v", err)
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
		UID:        bucketName,
		Address:    api.ParseCIDR("10.100.0.10/32"),
		ListenPort: 51820,
		PrivateKey: &privkey,
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
	if err := c.WireguardServerConfig().Create(wgsc); err != nil {
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

// func TestWireguardServerAddClient(t *testing.T) {
// 	var (
// 		bucketName  = "wg-dev"
// 		resourceUID = "user@domain.tld"
// 	)
// 	e, err := pgputils.NewEntity("user", "a user entity", resourceUID)
// 	if err != nil {
// 		t.Fatalf("failed generating new entity: %v", err)
// 	}

// 	clientPrivkey, err := api.GeneratePrivateKey()
// 	if err != nil {
// 		t.Fatalf("failed generating client private key: %v", err)
// 	}
// 	wgcc := &api.WireguardClientConfig{
// 		UID: resourceUID,
// 		InterfaceClientConfig: api.InterfaceClientConfig{
// 			PrivateKey: &clientPrivkey,
// 			Address:    api.ParseCIDR("192.168.0.100/32"),
// 			DNS:        []net.IP{net.ParseIP("1.1.1.1"), net.ParseIP("8.8.8.8")},
// 		},
// 		PeerClientConfig: api.PeerClientConfig{
// 			PublicKey:           clientPrivkey.PublicKey().String(),
// 			AllowedIPs:          api.ParseAllowedIPs("0.0.0.0/0", "::/0"),
// 			Endpoint:            "wg-dev.vpn.domain.tld:51820",
// 			PersistentKeepAlive: 25,
// 		},
// 	}

// 	expTunnelConfig, err := api.ParseWireguardClientConfigTemplate(wgcc)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	c := NewOrDie("", bucketName, &bolt.Options{OpenFile: openTempFile})
// 	serverPrivkey, err := api.GeneratePrivateKey()
// 	if err != nil {
// 		t.Fatalf("failed generating server private key: %v", err)
// 	}
// 	// this object will be validated when adding the client
// 	wgsc := &api.WireguardServerConfig{
// 		UID:        bucketName,
// 		Address:    api.ParseCIDR("10.100.0.10/32"),
// 		ListenPort: 51820,
// 		PrivateKey: &serverPrivkey,
// 		PostUp: []string{
// 			"ip link set mtu 1500 dev ens4",
// 			"ip link set mtu 1500 dev %i",
// 			"sysctl -w net.ipv4.ip_forward=1",
// 		},
// 		PostDown: []string{
// 			"sysctl -w net.ipv4.ip_forward=0",
// 			"iptables -D FORWARD -o %i -j ACCEPT",
// 			"iptables -D FORWARD -i %i -j ACCEPT",
// 		},
// 	}
// 	if err := c.WireguardServerConfig().Create(wgsc); err != nil {
// 		t.Fatalf("failed creating wg server config: %v", err)
// 	}
// 	encrypted, err := c.WireguardServerConfig().AddClient(bucketName, wgcc, []*openpgp.Entity{e})
// 	if err != nil {
// 		t.Fatalf("failed adding client to server: %v", err)
// 	}

// 	gotTunnelConfig, err := pgputils.ReadMessage(encrypted, e)
// 	if err != nil {
// 		t.Fatalf("failed reading message: %v", err)
// 	}
// 	if diff := cmp.Diff(expTunnelConfig, gotTunnelConfig); diff != "" {
// 		t.Fatalf("unexpected tunnel config (-want +got):\n%s", diff)
// 	}
// }
