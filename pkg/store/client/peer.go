package client

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"time"

	"github.com/sandromello/wgadmin/pkg/api"
	"github.com/sandromello/wgadmin/pkg/store"
)

// Peer methods to interact with store
type Peer interface {
	Get(name string) (*api.Peer, error)
	Update(obj *api.Peer) error
	Delete(name string) error
	List() ([]api.Peer, error)
	ListByServer(prefix string) ([]api.Peer, error)
	SearchByPubKey(server, pubkey string) (*api.Peer, error)
}

type peer struct {
	store  *store.Database
	prefix string
}

// Get retrieves a peer by its name
func (c *peer) Get(name string) (*api.Peer, error) {
	key := path.Join(c.prefix, name)
	data, err := c.store.Get(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var obj api.Peer
	return &obj, json.Unmarshal(data, &obj)
}

// SearchByPubKey find a peer by its public key
func (c *peer) SearchByPubKey(server, pubKey string) (*api.Peer, error) {
	_, err := api.ParseKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("invalid pubkey, %v", err)
	}
	peerList, err := c.ListByServer(server)
	if err != nil {
		return nil, err
	}
	for _, peer := range peerList {
		if peer.Status != api.PeerStatusActive {
			continue
		}
		if peer.PublicKeyString() == pubKey {
			return &peer, nil
		}
	}
	return nil, nil
}

// Update create or update a peer in the store
func (c *peer) Update(obj *api.Peer) error {
	obj.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.store.Set(path.Join(c.prefix, obj.UID), jsonData)
}

// Delete the object by its name
func (c *peer) Delete(name string) error {
	key := path.Join(c.prefix, name)
	return c.store.Del(key)
}

// List all the peer objects
func (c *peer) List() ([]api.Peer, error) {
	var peers []api.Peer
	return peers, c.store.Search(c.prefix, regexp.MustCompile(".*"), func(k, v []byte) error {
		var obj api.Peer
		if err := json.Unmarshal(v, &obj); err != nil {
			return err
		}
		peers = append(peers, obj)
		return nil
	})
}

// ListByServer all the peer objects from a given server
func (c *peer) ListByServer(server string) ([]api.Peer, error) {
	var peers []api.Peer
	pattern := fmt.Sprintf("%s/.+", server)
	serverPrefix := fmt.Sprintf("%s/%s", c.prefix, server)
	return peers, c.store.Search(serverPrefix, regexp.MustCompile(pattern), func(k, v []byte) error {
		var obj api.Peer
		if err := json.Unmarshal(v, &obj); err != nil {
			return err
		}
		peers = append(peers, obj)
		return nil
	})
}
