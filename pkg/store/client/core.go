package client

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sandromello/wgadmin/pkg/store"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	wgserverPrefix string = "/wgsconfig"
	wgclientPrefix string = "/wgcconfig"
	peerPrefix     string = "/peers"
	bucketName     string = "wireguard"
)

// Client objects to interact with store
type Client interface {
	WireguardClientConfig() WireguardClientConfig
	WireguardServerConfig() WireguardServerConfig
	Peer() Peer
	SyncGCS() error
	Close() error
}

type coreClient struct {
	wireguardClientConfig *wireguardClientConfig
	wireguardServerConfig *wireguardServerConfig
	peer                  *peer

	bucket string
}

// WireguardClientConfig creates a client to interact with wg client config
func (c *coreClient) WireguardClientConfig() WireguardClientConfig {
	return c.wireguardClientConfig
}

// WireguardServerConfig creates a client to interact with wg server config
func (c *coreClient) WireguardServerConfig() WireguardServerConfig {
	return c.wireguardServerConfig
}

func (c *coreClient) Peer() Peer {
	return c.peer
}

func (c *coreClient) Close() error {
	return c.peer.store.Close()
}

func (c *coreClient) SyncGCS() error {
	dbfile := c.peer.store.Path()
	if err := c.Close(); err != nil {
		return err
	}
	d := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()
	creds, err := google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)
	if err != nil {
		log.Fatal(err)
	}
	storageClient, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return err
	}
	gcsBucketName, err := getBucketEnvName()
	if err != nil {
		return err
	}
	w := storageClient.Bucket(gcsBucketName).
		Object(store.DBFileName).
		NewWriter(ctx)
	f, err := os.Open(dbfile)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = io.Copy(w, f); err != nil {
		return err
	}
	return w.Close()
}

// New initializes the store or returns an error
func New(dbfile, bucket string, opts *bolt.Options) (Client, error) {
	db, err := store.New(dbfile, bucketName, opts)
	if err != nil {
		return nil, err
	}
	return &coreClient{
		wireguardClientConfig: &wireguardClientConfig{
			store:  db,
			prefix: wgclientPrefix,
		},
		wireguardServerConfig: &wireguardServerConfig{
			store:  db,
			prefix: wgserverPrefix,
		},
		peer: &peer{
			store:  db,
			prefix: peerPrefix,
		},
	}, db.CreateBucketIfNotExists(bucketName)
}

// NewOrDie initializes the store or die (panic)
func NewOrDie(dbfile, bucket string, opts *bolt.Options) Client {
	c, err := New(dbfile, bucket, opts)
	if err != nil {
		panic(err)
	}
	return c
}

func getBucketEnvName() (string, error) {
	gcsBucketName := os.Getenv("GCS_BUCKET_NAME")
	if gcsBucketName == "" {
		return "", fmt.Errorf("GCS_BUCKET_NAME env not set or empty")
	}
	return gcsBucketName, nil
}

func fetchFromGCS(path string, flag int, mode os.FileMode) (*os.File, error) {
	d := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	creds, err := google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)
	if err != nil {
		log.Fatal(err)
	}
	storageClient, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	gcsBucketName, err := getBucketEnvName()
	if err != nil {
		return nil, err
	}
	rc, err := storageClient.Bucket(gcsBucketName).
		Object(store.DBFileName).
		NewReader(ctx)
	if err != nil && strings.Contains(err.Error(), "object doesn't exist") {
		return os.Create(path)
	}
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := ioutil.ReadAll(rc)
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	_, err = f.Write(data)
	return f, err
}

// NewGCS initializes the store from a Google Cloud Storage bucket
func NewGCS(dbfile string) (Client, error) {
	return New(dbfile, "", &bolt.Options{OpenFile: fetchFromGCS})
}
