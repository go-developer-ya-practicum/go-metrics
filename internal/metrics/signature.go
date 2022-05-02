package metrics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func Sign(metric *Metric, key string) (err error) {
	h, err := computeHash(metric, key)
	if err != nil {
		return err
	}

	metric.Hash = hex.EncodeToString(h)
	return nil
}

func Validate(metric *Metric, key string) (bool, error) {
	computed, err := computeHash(metric, key)
	if err != nil {
		return false, err
	}

	decoded, err := hex.DecodeString(metric.Hash)
	if err != nil {
		return false, err
	}

	return hmac.Equal(computed, decoded), nil
}

func computeHash(metric *Metric, key string) ([]byte, error) {
	var msg string
	switch metric.MType {
	case CounterType:
		msg = fmt.Sprintf("%s:%s:%d", metric.ID, metric.MType, *metric.Delta)
	case GaugeType:
		msg = fmt.Sprintf("%s:%s:%f", metric.ID, metric.MType, *metric.Value)
	default:
		return nil, fmt.Errorf("unknown metric type")
	}

	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(msg))
	return h.Sum(nil), nil
}
