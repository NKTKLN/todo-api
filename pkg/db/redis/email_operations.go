package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/NKTKLN/todo-api/models"
)

func (c *RedisClients) AddEmailData(ctx context.Context, data interface{}) (key string, err error) {
	// Create a temporary access key to verify mail fidelity and then send it
	key = uuid.NewString()
	err = c.EmailClient.Set(ctx, key, data, 15*time.Minute).Err()
	if err != nil {
		return
	}
	return
}

func (c *RedisClients) GetEmailData(ctx context.Context, key string) (val string, err error) {
	return c.EmailClient.Get(ctx, key).Result()
}

func (c *RedisClients) GetUserData(ctx context.Context, key string) (userParam models.Users, err error) {
	// Checking that the key is in working order
	val, err := c.EmailClient.Get(ctx, key).Result()
	if err != nil {
		return
	}

	// Converting user data from json
	err = json.Unmarshal([]byte(val), &userParam)
	return
}

func (c *RedisClients) DeleteEmailData(ctx context.Context, key string) (err error) {
	pipe := c.EmailClient.Pipeline()
	pipe.Del(ctx, key)
	_, err = pipe.Exec(ctx)
	return 
}
