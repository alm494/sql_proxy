package db

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

func (o DbConnInfo) GetHash() ([32]byte, error) {
	var buf bytes.Buffer
	var hash [32]byte

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(o)
	if err != nil {
		return hash, err
	}

	hash = sha256.Sum256(buf.Bytes())
	return hash, nil
}
