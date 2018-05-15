package db

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
)

// This won't survive in the longterm, we will need a RDB but for now just for playing state this will suffice
type DB struct {
	db   *leveldb.DB
	path string
}

func (self *DB) Put(key []byte, value []byte) error {
	err := self.db.Put(key, value, nil)
	return err
}

func (self *DB) Get(key []byte) ([]byte, error) {
	data, err := self.db.Get(key, nil)
	return data, err
}

func (self *DB) Close() {
	err := self.db.Close()
	if err == nil {
		fmt.Println("Database closed")
	} else {
		fmt.Println("Failed to close database", "err", err)
	}
}

func NewDb(file string) (*DB, error) {
	fmt.Println("OpeningDB at", file)
	db, err := leveldb.OpenFile(file, nil)
	ldb := &DB{db: db, path: file}
	return ldb, err
}
