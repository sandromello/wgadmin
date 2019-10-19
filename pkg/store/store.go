package store

import (
	"bytes"
	"regexp"

	bolt "go.etcd.io/bbolt"
)

// DBFileName is the name of the database file
const DBFileName string = "wireguard.db"

type AppendFn func(k, v []byte) error
type TransactionFn func(tx *bolt.Tx) error

// Database holds a database connection
type Database struct {
	db     *bolt.DB
	bucket string
}

// Close the connection with the database
func (d *Database) Close() error {
	return d.db.Close()
}

// GetBucket retrieves the bucket name
func (d *Database) GetBucket() string {
	return d.bucket
}

// NewInstance returns a new database instance with a new bucket
func (d *Database) NewInstance(b string) *Database {
	return &Database{db: d.db, bucket: b}
}

// CreateBucketIfNotExists create a bucket if doesn't exists
func (d *Database) CreateBucketIfNotExists(b string) error {
	// If the bucket is empty, don't do nothing
	if b == "" {
		return nil
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(b))
		return err
	})
}

// ListBuckets returns all buckets from bolt db
func (d *Database) ListBuckets() ([]string, error) {
	var buckets []string
	return buckets, d.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			buckets = append(buckets, string(name))
			return nil
		})
	})
}

// Path return the path of the current open database
func (d *Database) Path() string {
	return d.db.Path()
}

// Set writes data to a specified key
func (d *Database) Set(key string, data []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(d.bucket))
		if b == nil {
			// return fmt.Errorf("bucket %q doesn't exists", d.bucket)
			return nil
		}
		return b.Put([]byte(key), data)
	})
}

// Get a data from a given key
func (d *Database) Get(key string) ([]byte, error) {
	var data []byte
	return data, d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(d.bucket))
		if b == nil {
			// return fmt.Errorf("bucket %q doesn't exists", d.bucket)
			return nil
		}
		data = b.Get([]byte(key))
		return nil
	})
}

// Transaction manipulates a bolt transaction
func (d *Database) Transaction(t func(tx *bolt.Tx) error) error {
	return d.db.Update(t)
}

// DelBucket delete a bucket
func (d *Database) DelBucket(name string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(name))
	})
}

// Del delete a key
func (d *Database) Del(key string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(d.bucket))
		if b == nil {
			// return fmt.Errorf("bucket %q doesn't exists", d.bucket)
			return nil
		}
		return b.Delete([]byte(key))
	})
}

// Search will seek for a given key based on a prefix and a regular expression
func (d *Database) Search(prefix string, re *regexp.Regexp, appendFn AppendFn) error {
	return d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(d.bucket))
		if b == nil {
			// return fmt.Errorf("bucket %q doesn't exists", d.bucket)
			return nil
		}
		c := b.Cursor()
		pfx := []byte(prefix)
		for k, v := c.Seek(pfx); k != nil && bytes.HasPrefix(k, pfx) && re.Match(k); k, v = c.Next() {
			if err := appendFn(k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// New creates a new database
func New(dbfile, bucket string, opts *bolt.Options) (*Database, error) {
	db, err := bolt.Open(dbfile, 0600, opts)
	if err != nil {
		return nil, err
	}
	conn := &Database{db, bucket}
	return conn, nil
	// return conn, conn.CreateBucketIfNotExists(bucket)
}

// NewOrDie create a new database or panics
func NewOrDie(dbfile, bucket string, opts *bolt.Options) *Database {
	s, err := New(dbfile, bucket, opts)
	if err != nil {
		panic(err)
	}
	return s
}
