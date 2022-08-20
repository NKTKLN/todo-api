package redis

import (
	"context"

	"github.com/go-redis/redis/v8"

	"github.com/NKTKLN/todo-api/pkg/db"
)

type RedisClients struct {
	EmailClient        *redis.Client
	AccessTokenClient  *redis.Client
	RefreshTokenClient *redis.Client
}

// Connecting to a redis database
func Connect(redisAddr, redisPassword string, redisEmailDB, redisAccessTokenDB, redisRefreshTokenDB int) (db.RedisClient, error) {
	var ctx = context.Background()
	
	emailClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisEmailDB,
	})
	if err := emailClient.Ping(ctx).Err(); err != nil {
		return &RedisClients{}, err
	}
	accessTokenClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisAccessTokenDB,
	})
	if err := accessTokenClient.Ping(ctx).Err(); err != nil {
		return &RedisClients{}, err
	}
	refreshTokenClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisRefreshTokenDB,
	})
	if err := refreshTokenClient.Ping(ctx).Err(); err != nil {
		return &RedisClients{}, err
	}

	return &RedisClients{
		EmailClient:        emailClient,
		AccessTokenClient:  accessTokenClient,
		RefreshTokenClient: refreshTokenClient,
	}, nil
}
