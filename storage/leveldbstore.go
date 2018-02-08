package storage

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
)

type Handler interface {
	CloseKVSHandler()
	PutKVS(batch map[string][]byte) error
	GetPayloadData(topic string) ([]byte, error)
}

type KVSHandler struct {
	kvs *leveldb.DB
}

func NewMsgKvsHandler(path string) Handler {
	kvs, err := leveldb.OpenFile(path, nil)
	if err != nil {
		log.Error("storage/leveldb: openfile error:", err)
		panic(err)
	}

	h := KVSHandler{
		kvs: kvs,
	}

	return &h
}

func (h *KVSHandler) CloseKVSHandler() {
	h.kvs.Close()
}

func (h *KVSHandler) PutKVS(batch map[string][]byte) error {
	for key, value := range batch {
		err := h.kvs.Put([]byte(key), value, nil)
		if err != nil {
			log.Error("storage/leveldb: Put error:", err)
			return err
		}
	}
	return nil
}

func (h *KVSHandler) GetPayloadData(topic string) ([]byte, error) {
	data, err := h.kvs.Get([]byte(topic), nil)
	if err != nil {
		log.Error("storage/leveldb: get error:", err)
		return nil, err
	}
	return data, nil
}
