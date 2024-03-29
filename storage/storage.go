package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"

	"time"

	"github.com/juju/mutex"
	"modernc.org/kv"
)

type fakeClock struct {
	delay time.Duration
}

func (f *fakeClock) After(time.Duration) <-chan time.Time {
	return time.After(f.delay)
}

func (f *fakeClock) Now() time.Time {
	return time.Now()
}

type filelocker struct {
	releaser mutex.Releaser
}

func (lock *filelocker) Close() error {
	lock.releaser.Release()
	return nil
}

// helper
func exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Storage represented a key-value store backed by a disk file
type Storage struct {
	db   *kv.DB
	name string
}

var (
	// stores open Storage instances (map from name to Storage)
	openDBs map[string]*Storage = make(map[string]*Storage)
)

// New returns a pointer to the store and err == nil.
//
// On error, err != nil.
//
// New can be called whether the named store already exists or not (
// in the latter case, it will be created)
func New(name string) (storage *Storage, err error) {
	var (
		exist  bool
		isOpen bool
		db     *kv.DB
	)

	if storage, isOpen = openDBs[name]; isOpen {
		return storage, nil
	}

	exist, err = exists(name)
	if err != nil {
		return nil, err
	}

	options := &kv.Options{
		Locker: func(name string) (io.Closer, error) {
			var (
				fpath    string
				spec     mutex.Spec
				err      error
				locker   filelocker
				h        hash.Hash
				sha1Hash string
				releaser mutex.Releaser
			)
			fpath, err = filepath.Abs(name)
			h = sha1.New()
			h.Write([]byte(fpath))
			sha1Hash = hex.EncodeToString(h.Sum(nil))
			if err != nil {
				panic(err)
			}
			spec = mutex.Spec{
				Name:    "m" + sha1Hash[0:38],
				Clock:   &fakeClock{delay: time.Second},
				Delay:   time.Millisecond,
				Timeout: time.Second,
			}
			locker = filelocker{}
			releaser, err = mutex.Acquire(spec)
			locker.releaser = releaser
			if err != nil {
				return nil, err
			}
			return &locker, nil
		},
	}

	if exist {
		db, err = kv.Open(name, options)
	} else {
		db, err = kv.Create(name, options)
	}

	if err != nil {
		return nil, err
	}

	storage = &Storage{name: name, db: db}

	openDBs[name] = storage
	return storage, nil

}

// Close closes a store. It is idempotent
func (storage *Storage) Close() {
	storage.db.Close()
	if _, isOpen := openDBs[storage.name]; isOpen {
		delete(openDBs, storage.name)
	}
}

// Clear both calls Close() and erases the disk file associated to the storage.
func (storage *Storage) Clear() {
	if _, isOpen := openDBs[storage.name]; isOpen {
		storage.Close()
		os.Remove(storage.name)
	}
}

// Get returns the value associated with key, or an empty slice if no such value exists.
// The returned slice may be a sub-slice of buf if buf was large enough to hold
// the entire content. Otherwise, a newly allocated slice will be returned. It
// is valid to pass a nil buf.
//
// Get is atomic and it is safe for concurrent use by multiple goroutines.
func (storage *Storage) Get(buf, key []byte) ([]byte, error) {
	return storage.db.Get(buf, key)
}

// Set sets the value associated with key.
// Any previous value, if existed, is overwritten by the new one.
//
// Set is atomic and it is safe for concurrent use by multiple goroutines.
func (storage *Storage) Set(key, value []byte) error {
	return storage.db.Set(key, value)
}

// Delete deletes key and its associated value from the DB.
//
// Delete is atomic and it is safe for concurrent use by multiple goroutines.
func (storage *Storage) Delete(key []byte) error {
	return storage.db.Delete(key)
}

// BeginTransaction starts a new transaction. Every call to BeginTransaction must
// be eventually "balanced" by exactly one call to Commit or Rollback (but not both).
// Calls to BeginTransaction may nest.
//
// BeginTransaction is atomic and it is safe for concurrent use by multiple goroutines
// (if/when that makes sense).
func (storage *Storage) BeginTransaction() error {
	return storage.db.BeginTransaction()
}

// Commit commits the current transaction. If the transaction is the
// top level one, then all of the changes made within the transaction
// are atomically made persistent in the DB. Invocation of an unbalanced Commit is an error.
//
// Commit is atomic and it is safe for concurrent use by multiple goroutines
// (if/when that makes sense).
func (storage *Storage) Commit() error {
	return storage.db.Commit()
}

// Rollback cancels and undoes the innermost transaction level. If the
// transaction is the top level one, then no of the changes made within the
// transactions are persisted. Invocation of an unbalanced Rollback is an
// error.
//
// Rollback is atomic and it is safe for concurrent use by multiple goroutines
// (if/when that makes sense).
func (storage *Storage) Rollback() error {
	return storage.db.Rollback()
}
