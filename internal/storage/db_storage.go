package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"

	"github.com/hikjik/go-musthave-devops-tpl.git/internal/config"
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

func (s *DBStorage) PutGauge(name string, value float64) {
	_, err := s.Exec(
		context.Background(),
		"INSERT INTO gauge (name, value) VALUES ($1, $2) ON CONFLICT(name) DO UPDATE SET value = $2;",
		name, value)
	if err != nil {
		log.Warnf("Failed to put metric: %v", err)
	}
}

func (s *DBStorage) UpdateCounter(name string, value int64) {
	_, err := s.Exec(
		context.Background(),
		"INSERT INTO counter (name, value) VALUES ($1, $2) ON CONFLICT(name) DO UPDATE SET value = counter.value + $2;",
		name, value)
	if err != nil {
		log.Warnf("Failed to put metric: %v", err)
	}
}

func (s *DBStorage) GetGauge(name string) (value float64, ok bool) {
	row := s.QueryRow(context.Background(), "SELECT value FROM gauge WHERE name=$1;", name)

	switch err := row.Scan(&value); err {
	case sql.ErrNoRows:
		ok = false
	case nil:
		ok = true
	default:
		ok = false
		log.Warnf("Failed to query db: %v", err)
	}
	return
}

func (s *DBStorage) GetCounter(name string) (value int64, ok bool) {
	row := s.QueryRow(context.Background(), "SELECT value FROM counter WHERE name=$1;", name)

	switch err := row.Scan(&value); err {
	case sql.ErrNoRows:
		ok = false
	case nil:
		ok = true
	default:
		ok = false
		log.Warnf("Failed to query db: %v", err)
	}
	return
}

func (s *DBStorage) GetGaugeMetrics() map[string]float64 {
	metrics := make(map[string]float64)

	var name string
	var value float64
	rows, err := s.Query(context.Background(), "SELECT name, value FROM gauge")
	if err != nil {
		log.Warnf("Failed to query db: %v", err)
		return nil
	}
	for rows.Next() {
		if err = rows.Scan(&name, &value); err != nil {
			log.Warnf("Failed to query db: %v", err)
			return nil
		}
		metrics[name] = value
	}
	if err = rows.Err(); err != nil {
		log.Warnf("Failed to query db: %v", err)
		return nil
	}
	return metrics
}

func (s *DBStorage) GetCounterMetrics() map[string]int64 {
	metrics := make(map[string]int64)

	var name string
	var value int64
	rows, err := s.Query(context.Background(), "SELECT name, value FROM counter")
	if err != nil {
		log.Warnf("Failed to query db: %v", err)
		return nil
	}
	for rows.Next() {
		if err = rows.Scan(&name, &value); err != nil {
			log.Warnf("Failed to query db: %v", err)
			return nil
		}
		metrics[name] = value
	}
	if err = rows.Err(); err != nil {
		log.Warnf("Failed to query db: %v", err)
		return nil
	}
	return metrics
}
