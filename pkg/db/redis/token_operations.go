package redis

import (
	"context"
	"strconv"

	"github.com/NKTKLN/todo-api/models"
	"github.com/NKTKLN/todo-api/pkg/common"
	"github.com/spf13/viper"
)

func (c *RedisClients) CreateTokens(ctx context.Context, userId int) (accessToken, refreshToken string, err error) {
	// Generation of refreshes and access tokens
	accessToken, err = common.NewJWT(userId, models.ACCESS_TOKEN_LIVE, viper.GetString("api.jwt.access-secret"))
	if err != nil {
		return
	}
	refreshToken, err = common.NewJWT(userId, models.ACCESS_TOKEN_LIVE, viper.GetString("api.jwt.refresh-secret"))
	if err != nil {
		return
	}

	// Adding tokens to the db
	if err = c.AccessTokenClient.Set(ctx, strconv.Itoa(userId), accessToken, models.ACCESS_TOKEN_LIVE).Err(); err != nil {
		return
	}
	if err = c.RefreshTokenClient.Set(ctx, strconv.Itoa(userId), accessToken, models.REFRESH_TOKEN_LIVE).Err(); err != nil {
		return
	}

	return
}

func (c *RedisClients) CheckTokens(ctx context.Context, userId int) (accessToken, refreshToken string, err error) {
	// Check if the access token exists in the db if not, then create it
	accessToken, _ = c.AccessTokenClient.Get(ctx, strconv.Itoa(userId)).Result()
	if accessToken == "" {
		accessToken, err = common.NewJWT(userId, models.ACCESS_TOKEN_LIVE, viper.GetString("api.jwt.access-secret"))
		if err != nil {
			return
		}

		if err = c.AccessTokenClient.Set(ctx, strconv.Itoa(userId), accessToken, models.ACCESS_TOKEN_LIVE).Err(); err != nil {
			return
		}
	}

	// Check if the refresh token exists in the db, and if not, create it
	refreshToken, _ = c.RefreshTokenClient.Get(ctx, strconv.Itoa(userId)).Result()
	if refreshToken == "" {
		refreshToken, err = common.NewJWT(userId, models.REFRESH_TOKEN_LIVE, viper.GetString("api.jwt.refresh-secret"))
		if err != nil {
			return
		}

		if err = c.RefreshTokenClient.Set(ctx, strconv.Itoa(userId), refreshToken, models.REFRESH_TOKEN_LIVE).Err(); err != nil {
			return
		}

	}

	return
}

func (c *RedisClients) VerifyToken(ctx context.Context, tokenString string) int {
	// Check the validity of the token and get the user id from it
	userId := common.VerifyToken(tokenString, viper.GetString("api.jwt.access-secret"))
	if userId == 0 {
		return 0
	}

	// Token validity check
	dbToken := c.AccessTokenClient.Get(ctx, strconv.Itoa(userId)).Val()
	if dbToken != tokenString {
		return 0
	}

	// Renewing the life of a token
	if c.AccessTokenClient.Set(ctx, strconv.Itoa(userId), tokenString, models.ACCESS_TOKEN_LIVE).Err() != nil {
		return 0
	}

	return userId
}

func (c *RedisClients) VerifyRefreshToken(ctx context.Context, tokenString string) int {
	// Check the validity of the token and get the user id from it
	userId := common.VerifyToken(tokenString, viper.GetString("api.jwt.refresh-secret"))
	if userId == 0 {
		return 0
	}

	// Token validity check
	dbToken := c.RefreshTokenClient.Get(ctx, strconv.Itoa(userId)).Val()
	if dbToken != tokenString {
		return 0
	}

	// Renewing the life of a token
	if c.RefreshTokenClient.Set(ctx, strconv.Itoa(userId), tokenString, models.REFRESH_TOKEN_LIVE).Err() != nil {
		return 0
	}

	return userId
}

func (c *RedisClients) DeleteRefreshTokensData(ctx context.Context, userId int) (err error) {
	refreshTokenPipe := c.RefreshTokenClient.Pipeline()
	refreshTokenPipe.Del(ctx, strconv.Itoa(userId))
	_, err = refreshTokenPipe.Exec(ctx)
	return 
}

func (c *RedisClients) DeleteAccessTokensData(ctx context.Context, userId int) (err error) {
	accessTokenPipe := c.AccessTokenClient.Pipeline()
	accessTokenPipe.Del(ctx, strconv.Itoa(userId))
	_, err = accessTokenPipe.Exec(ctx)
	return 
}
