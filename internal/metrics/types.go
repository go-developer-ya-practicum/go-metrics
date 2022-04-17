package metrics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func (m *Metrics) SetHash(key string) (err error) {
	sign, err := m.computeHash(key)
	if err != nil {
		return err
	}

	m.Hash = hex.EncodeToString(sign)
	return nil
}

func (m *Metrics) ValidateHash(key string) (bool, error) {
	computed, err := m.computeHash(key)
	if err != nil {
		return false, err
	}

	decoded, err := hex.DecodeString(m.Hash)
	if err != nil {
		return false, err
	}

	return hmac.Equal(computed, decoded), nil
}

func (m *Metrics) computeHash(key string) ([]byte, error) {
	var msg string
	switch m.MType {
	case "counter":
		msg = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	case "gauge":
		msg = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	default:
		return nil, fmt.Errorf("unknown metric type")
	}

	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(msg))
	return h.Sum(nil), nil
}
