package metrics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// Signer интерфейс, предоставляющий механизм подписи передаваемых метрик
type Signer interface {
	// Sign создает подпись для указанной метрики
	Sign(metric *Metric) error

	// Validate производит проверку подписи для указанной метрики
	Validate(metric *Metric) (bool, error)
}

// hmacSigner подписывает значения метрик с помощью алгоритма HMAC
type hmacSigner struct {
	key []byte
}

// NewHMACSigner возвращает объект hmacSigner
func NewHMACSigner(key string) *hmacSigner {
	if key == "" {
		return nil
	}
	return &hmacSigner{
		key: []byte(key),
	}
}

// Sign вычисляет подпись для метрики по алгоритму SHA256
// и сохраняет в поле Hash значение полученного хеш-значения.
func (s *hmacSigner) Sign(metric *Metric) error {
	if s == nil {
		return nil
	}
	h, err := s.computeHash(metric)
	if err != nil {
		return err
	}

	metric.Hash = hex.EncodeToString(h)
	return nil
}

// Validate проверяет валидность хеш-значения метрики
func (s *hmacSigner) Validate(metric *Metric) (bool, error) {
	if s == nil {
		return true, nil
	}
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

func (s *hmacSigner) computeHash(metric *Metric) ([]byte, error) {
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
	if _, err := io.WriteString(h, msg); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
