package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStorage exercises the GetStore method
func TestStorage(t *testing.T) {
	var (
		err     error
		storage *Storage
		data    []byte
	)

	// try to create
	if storage, err = New("/tmp/raftdb"); err != nil {
		t.Fatal(err)
	}

	defer storage.Clear()

	// close
	storage.Close()

	// try to open
	if storage, err = New("/tmp/raftdb"); err != nil {
		t.Fatal(err)
	}

	defer storage.Close()

	if _, err = New("/tmp/raftdb"); err != nil {
		t.Fatal(err)
	}

	// test retrieval
	if err = storage.Set([]byte("somekey"), []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}

	if data, err = storage.Get(nil, []byte("somekey")); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []byte{1, 2, 3}, data)

	if err = storage.BeginTransaction(); err != nil {
		t.Fatal(err)
	}

	if err = storage.Set([]byte("somekey"), []byte("before rollback")); err != nil {
		t.Fatal(err)
	}

	if err = storage.Rollback(); err != nil {
		t.Fatal(err)
	}

	storage.Close()
	if storage, err = New("/tmp/raftdb"); err != nil {
		t.Fatal(err)
	}
	defer storage.Clear()

	if data, err = storage.Get(nil, []byte("somekey")); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []byte{1, 2, 3}, data)

}
