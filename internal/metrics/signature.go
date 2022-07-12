package metrics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Signer подписывает значения метрик с помощью алгоритма HMAC
type Signer struct {
	key []byte
}

// NewSigner возвращает новый Signer
func NewSigner(key string) *Signer {
	if key == "" {
		return nil
	}
	return &Signer{
		key: []byte(key),
	}
}

// Sign вычисляет подпись для метрики по алгоритму SHA256
// и сохраняет в поле Hash значение полученного хеш-значения.
func (s *Signer) Sign(metric *Metric) error {
	h, err := s.computeHash(metric)
	if err != nil {
		return err
	}

	metric.Hash = hex.EncodeToString(h)
	return nil
}

// Validate проверяет валидность хеш-значения метрики
func (s *Signer) Validate(metric *Metric) (bool, error) {
	computed, err := s.computeHash(metric)
	if err != nil {
		return false, err
	}

	decoded, err := hex.DecodeString(metric.Hash)
	if err != nil {
		return false, err
	}

	return hmac.Equal(computed, decoded), nil
}

func (s *Signer) computeHash(metric *Metric) ([]byte, error) {
	var msg string
	switch metric.MType {
	case CounterType:
		msg = fmt.Sprintf("%s:%s:%d", metric.ID, metric.MType, *metric.Delta)
	case GaugeType:
		msg = fmt.Sprintf("%s:%s:%f", metric.ID, metric.MType, *metric.Value)
	default:
		return nil, fmt.Errorf("unknown metric type")
	}

	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(msg))
	return h.Sum(nil), nil
}
