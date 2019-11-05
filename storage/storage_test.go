package storage

import (
	"os"
	"testing"
)

// TestStorage exercises the GetStore method
func TestStorage(t *testing.T) {
	// delete db
	os.Remove("/tmp/raftdb")

	var (
		err     error
		storage *Storage
	)

	// try to create
	storage, err = New("/tmp/raftdb")
	defer storage.Clear()
	if err != nil {
		t.Fatal(err)
	}

	// close
	storage.Close()
	if err != nil {
		t.Fatal(err)
	}

	// try to open
	storage, err = New("/tmp/raftdb")
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	_, err = New("/tmp/raftdb")
	if err != nil {
		t.Fatal(err)
	}

}
