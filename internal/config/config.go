// Package config предназначен для настройки агентов и сервера по сбору рантайм-метрик
package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog/log"
)

// AgentConfig содержит настройки агента по сбору метрик
type AgentConfig struct {
	Address        string        `env:"ADDRESS" json:"address"`
	SignatureKey   string        `env:"KEY" json:"key"`
	PublicKeyPath  string        `env:"CRYPTO_KEY" json:"crypto_key"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" json:"poll_interval"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" json:"report_interval"`
}

// StorageConfig содержит настройки хранилища метрик
type StorageConfig struct {
	StoreFile     string        `env:"STORE_FILE"`
	DatabaseDNS   string        `env:"DATABASE_DSN"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	Restore       bool          `env:"RESTORE"`
}

// ServerConfig содержит настройки сервера по сбору рантайм-метрик
type ServerConfig struct {
	Address           string `env:"ADDRESS"`
	SignatureKey      string `env:"KEY"`
	EncryptionKeyPath string `env:"CRYPTO_KEY"`
	StorageConfig     StorageConfig
}

// GetAgentConfig возвращает настройки AgentConfig
func GetAgentConfig() AgentConfig {
	var config AgentConfig
	var path string

	flag.StringVar(&config.Address, "a", "127.0.0.1:8080", "Server address")
	flag.DurationVar(&config.PollInterval, "p", time.Second*2, "Poll interval, sec")
	flag.DurationVar(&config.ReportInterval, "r", time.Second*10, "Report interval, sec")
	flag.StringVar(&config.SignatureKey, "k", "", "HMAC key")
	flag.StringVar(&config.PublicKeyPath, "crypto-key", "", "Path to public RSA key")
	flag.StringVar(&path, "c", "", "Path to json config file")
	flag.StringVar(&path, "config", "", "Path to json config file")
	flag.Parse()

	if err := parseConfigJSON(&config, path); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse agent json config")
	}

	// second call for correct priority
	flag.Parse()

	if err := env.Parse(&config); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse agent config")
	}

	return config
}

// GetServerConfig возвращает настройки ServerConfig
func GetServerConfig() ServerConfig {
	var config ServerConfig
	var path string

	flag.StringVar(&config.Address, "a", "127.0.0.1:8080", "Server Address")
	flag.StringVar(&config.SignatureKey, "k", "", "HMAC key")
	flag.StringVar(&config.StorageConfig.StoreFile, "f", "/tmp/devops-metrics-db.json", "Store File")
	flag.DurationVar(&config.StorageConfig.StoreInterval, "i", time.Second*300, "Store Interval")
	flag.BoolVar(&config.StorageConfig.Restore, "r", true, "Restore After Start")
	flag.StringVar(&config.StorageConfig.DatabaseDNS, "d", "", "Database DNS")
	flag.StringVar(&config.EncryptionKeyPath, "crypto-key", "", "Path to private RSA key")
	flag.StringVar(&path, "c", "", "Path to json config file")
	flag.StringVar(&path, "config", "", "Path to json config file")
	flag.Parse()

	if err := parseConfigJSON(&config, path); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse server json config")
	}

	// second call for correct priority
	flag.Parse()

	if err := env.Parse(&config); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse server config")
	}

	return config
}

func parseConfigJSON(cfg interface{}, path string) error {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err = json.Unmarshal(data, cfg); err != nil {
			return err
		}
	}
	return nil
}
