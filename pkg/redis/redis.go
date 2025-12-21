package redis

import (
	"log"
	"sql-service/configs"

	"context"

	"github.com/go-redis/redis/v8"
)

type Redisdb struct {
	client *redis.Client
}

func NewRedis(conf *configs.Config) *Redisdb {
	rdb := redis.NewClient(&redis.Options{})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	return &Redisdb{client: rdb}
}
