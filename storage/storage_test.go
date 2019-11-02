package storage

import (
	"os"
	"testing"
	"modernc.org/kv"
)

// TestStorage exercises the GetStore method
func TestStorage(t *testing.T) {
	// delete db
	os.Remove("/tmp/raftdb")

	var (
		err error
		db *kv.DB
	)

	// try to create
	db, err = GetStore("/tmp/raftdb")
	if err != nil {
		t.Error(err)
		return
	}

	// close to remove the lockfiles
	err = db.Close()
	if err != nil {
		t.Error(err)
		return
	}

	// try to open (because already created)
	_, err = GetStore("/tmp/raftdb")
	if err != nil {
		t.Error(err)
		return
	}


}
