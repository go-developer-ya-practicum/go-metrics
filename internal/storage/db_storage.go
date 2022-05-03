package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/openlyinc/pointy"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/config"
	"github.com/hikjik/go-musthave-devops-tpl.git/internal/metrics"
)

type DBStorage struct {
	*pgx.Conn
}

func newDBStorage(ctx context.Context, cfg config.StorageConfig) (Storage, error) {
	conn, err := pgx.Connect(ctx, cfg.DatabaseDNS)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS counter(
		name VARCHAR(128) PRIMARY KEY UNIQUE NOT NULL,
		value BIGINT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS gauge(
		name VARCHAR(128) PRIMARY KEY UNIQUE NOT NULL,
		value DOUBLE PRECISION NOT NULL
	);`)
	if err != nil {
		return nil, err
	}

	return &DBStorage{Conn: conn}, nil
}

func (s *DBStorage) Ping() error {
	if s.Conn == nil {
		return fmt.Errorf("failed to connect to db")
	}
	return s.Conn.Ping(context.Background())
}

func (s *DBStorage) Put(metric *metrics.Metric) error {
	switch metric.MType {
	case metrics.CounterType:
		if metric.Delta == nil {
			return ErrBadArgument
		}
		_, err := s.Exec(
			context.Background(),
			"INSERT INTO counter (name, value) "+
				"VALUES ($1, $2) "+
				"ON CONFLICT(name) DO UPDATE SET value = counter.value + $2;",
			metric.ID, *metric.Delta)
		return err
	case metrics.GaugeType:
		if metric.Value == nil {
			return ErrBadArgument
		}
		_, err := s.Exec(
			context.Background(),
			"INSERT INTO gauge (name, value) "+
				"VALUES ($1, $2) "+
				"ON CONFLICT(name) DO UPDATE SET value = $2;",
			metric.ID, *metric.Value)
		return err
	default:
		return ErrUnknownMetricType
	}
}

func (s *DBStorage) Get(metric *metrics.Metric) error {
	switch metric.MType {
	case metrics.CounterType:
		row := s.QueryRow(
			context.Background(),
			"SELECT value FROM counter WHERE name=$1;",
			metric.ID)

		var delta int64
		if err := row.Scan(&delta); err == nil {
			metric.Delta = pointy.Int64(delta)
			return nil
		} else {
			return ErrNotFound
		}
	case metrics.GaugeType:
		row := s.QueryRow(
			context.Background(),
			"SELECT value FROM gauge WHERE name=$1;",
			metric.ID)

		var value float64
		if err := row.Scan(&value); err == nil {
			metric.Value = pointy.Float64(value)
			return nil
		} else {
			return ErrNotFound
		}
	default:
		return ErrUnknownMetricType
	}
}

func (s *DBStorage) List() ([]*metrics.Metric, error) {
	result := make([]*metrics.Metric, 0)

	var (
		id    string
		value float64
		delta int64
	)

	rows, err := s.Query(context.Background(), "SELECT name, value FROM gauge")
	if err != nil {
		return nil, fmt.Errorf("failed to query db: %v", err)
	}
	for rows.Next() {
		if err = rows.Scan(&id, &value); err != nil {
			return nil, fmt.Errorf("failed to query db: %v", err)
		}
		result = append(result, metrics.NewGauge(id, value))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query db: %v", err)
	}

	rows, err = s.Query(context.Background(), "SELECT name, value FROM counter")
	if err != nil {
		return nil, fmt.Errorf("failed to query db: %v", err)
	}
	for rows.Next() {
		if err = rows.Scan(&id, &delta); err != nil {
			return nil, fmt.Errorf("failed to query db: %v", err)
		}
		result = append(result, metrics.NewCounter(id, delta))
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query db: %v", err)
	}

	return result, nil
}
