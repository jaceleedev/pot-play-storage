package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func NewDBPool(cfg map[string]interface{}) (*pgxpool.Pool, error) {
	host := cfg["host"].(string)
	user := cfg["user"].(string)
	password := cfg["password"].(string)
	name := cfg["name"].(string)
	sslMode := cfg["ssl_mode"].(string)

	// Safe type conversion for port
	var port int
	switch v := cfg["port"].(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case int64:
		port = int(v)
	default:
		return nil, fmt.Errorf("invalid port type: %T", v)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", user, password, host, port, name, sslMode)
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = 10
	poolCfg.MaxConnLifetime = 30 * time.Minute

	return pgxpool.NewWithConfig(context.Background(), poolCfg)
}

func NewRedis(cfg map[string]interface{}) *redis.Client {
	host := cfg["host"].(string)

	// Safe type conversion for port and db
	var port int
	switch v := cfg["port"].(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case int64:
		port = int(v)
	default:
		port = 6379 // default or error
	}

	var db int
	switch v := cfg["db"].(type) {
	case float64:
		db = int(v)
	case int:
		db = v
	case int64:
		db = int(v)
	default:
		db = 0 // default
	}

	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: "",
		DB:       db,
	})
}