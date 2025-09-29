package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

var RedisClient *redis.Client

// InitRedis initializes the Redis connection
func InitRedis() error {
	redisURL := viper.GetString("redis_url")
	if redisURL == "" {
		return fmt.Errorf("redis_url is not set in configuration")
	}

	// Parse Redis URL
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse redis URL: %v", err)
	}

	// Create Redis client
	RedisClient = redis.NewClient(opt)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %v", err)
	}

	log.Println("Redis connection established")
	return nil
}

// CloseRedis closes the Redis connection
func CloseRedis() error {
	if RedisClient != nil {
		log.Println("Closing Redis connection...")
		return RedisClient.Close()
	}
	return nil
}
