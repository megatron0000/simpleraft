package storage

import (
	"os"

	"modernc.org/kv"
)

func exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

var (
	openDBs map[string]*kv.DB = make(map[string]*kv.DB)
)

// GetStore returns a pointer to the store and err = nil.
//
// On error, err != nil.
//
// GetStore can be called whether the store already exists or not (
// in the latter case, it will be created)
//
// The store is implemented by modernc.org/kv.
// 
// You should not close the store
func GetStore(name string) (db *kv.DB, err error) {
	var (
		exist  bool
		isOpen bool
	)

	if db, isOpen = openDBs[name]; isOpen {
		return db, nil
	}

	exist, err = exists(name)
	if err != nil {
		return nil, err
	}

	if exist {
		db, err = kv.Open(name, &kv.Options{})
	} else {
		db, err = kv.Create(name, &kv.Options{})
	}

	if err != nil {
		return nil, err
	}

	openDBs[name] = db
	return db, nil

}
