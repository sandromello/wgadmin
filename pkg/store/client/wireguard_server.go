package client

import (
	"encoding/json"
	"path"
	"regexp"
	"time"

	"github.com/sandromello/wgadmin/pkg/api"
	"github.com/sandromello/wgadmin/pkg/store"
)

// WireguardServerConfig methods to interact with store
type WireguardServerConfig interface {
	Get(name string) (*api.WireguardServerConfig, error)
	Update(obj *api.WireguardServerConfig) error
	List() ([]api.WireguardServerConfig, error)
	Delete(name string) error
}

type wireguardServerConfig struct {
	store  *store.Database
	prefix string
}

// List all wireguard server config objects
func (w *wireguardServerConfig) List() ([]api.WireguardServerConfig, error) {
	var wgscList []api.WireguardServerConfig
	return wgscList, w.store.Search(w.prefix, regexp.MustCompile(".*"), func(k, v []byte) error {
		var obj api.WireguardServerConfig
		if err := json.Unmarshal(v, &obj); err != nil {
			return err
		}
		wgscList = append(wgscList, obj)
		return nil
	})
}

// Get retrieves a wireguard server config by its name
func (w *wireguardServerConfig) Get(name string) (*api.WireguardServerConfig, error) {
	key := path.Join(w.prefix, name)
	data, err := w.store.Get(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var obj api.WireguardServerConfig
	return &obj, json.Unmarshal(data, &obj)
}

// Update a new wireguard server config in the store, subsequent calls
// will override the object
func (w *wireguardServerConfig) Update(obj *api.WireguardServerConfig) error {
	obj.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	// if err := w.store.CreateBucketIfNotExists(obj.UID); err != nil {
	// 	return err
	// }
	return w.store.Set(path.Join(w.prefix, obj.UID), jsonData)
}

// Delete a wireguard server config by deleting the whole bucket
func (w *wireguardServerConfig) Delete(name string) error {
	return w.store.Del(path.Join(w.prefix, name))
}
