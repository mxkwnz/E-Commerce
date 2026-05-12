package cache

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var Client *redis.Client
var ctx = context.Background()

func Connect() error {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}

	Client = redis.NewClient(opt)

	if err := Client.Ping(ctx).Err(); err != nil {
		return err
	}

	log.Println("Connected to Redis")
	return nil
}

func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

func Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, expiration).Err()
}

func Get(key string, dest interface{}) error {
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func Delete(key string) error {
	return Client.Del(ctx, key).Err()
}

func DeletePattern(pattern string) error {
	iter := Client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := Client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}
