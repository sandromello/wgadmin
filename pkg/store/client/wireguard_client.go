package client

import (
	"encoding/json"
	"path"
	"regexp"

	"github.com/sandromello/wgadmin/pkg/api"
	"github.com/sandromello/wgadmin/pkg/store"
)

// WireguardClientConfig methods to interact with store
type WireguardClientConfig interface {
	Get(name string) (*api.WireguardClientConfig, error)
	Create(obj *api.WireguardClientConfig) error
	Delete(name string) error
	List() ([]api.WireguardClientConfig, error)
}

type wireguardClientConfig struct {
	store  *store.Database
	prefix string
}

// Get retrieves a wireguard client config by its name
func (c *wireguardClientConfig) Get(name string) (*api.WireguardClientConfig, error) {
	key := path.Join(c.prefix, name)
	data, err := c.store.Get(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var obj api.WireguardClientConfig
	return &obj, json.Unmarshal(data, &obj)
}

// Create a new wireguard client config in the store, subsequent calls
// will override the object
func (c *wireguardClientConfig) Create(obj *api.WireguardClientConfig) error {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.store.Set(path.Join(c.prefix, obj.UID), jsonData)
}

// Delete the object by its name
func (c *wireguardClientConfig) Delete(name string) error {
	key := path.Join(c.prefix, name)
	return c.store.Del(key)
}

// List all the wireguard client config objects
func (c *wireguardClientConfig) List() ([]api.WireguardClientConfig, error) {
	var wgClientConfigs []api.WireguardClientConfig
	return wgClientConfigs, c.store.Search(c.prefix, regexp.MustCompile(".*"), func(k, v []byte) error {
		var obj api.WireguardClientConfig
		if err := json.Unmarshal(v, &obj); err != nil {
			return err
		}
		wgClientConfigs = append(wgClientConfigs, obj)
		return nil
	})
}
