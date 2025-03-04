package dbs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/oauth2"
)

var TOKEN_MIN int = 5

type AppRedis struct {
	Redis *redis.Client
}

func InitRedisDb(context context.Context, DB_HOST string, DB_PASSWORD string) *AppRedis {
	redis := redis.NewClient(&redis.Options{
		Addr:     DB_HOST,
		Password: DB_PASSWORD,
		DB:       0,
	})
	return &AppRedis{Redis: redis}
}
func (r *AppRedis) SetSessionToken(ctx context.Context, key string, oauthToken *oauth2.Token) error {
	return r.Redis.Set(ctx, key, oauthToken, time.Duration(TOKEN_MIN)*time.Minute).Err()
}

func (r *AppRedis) GetSessionToken(ctx context.Context, key string) (string, error) {
	return r.Redis.Get(ctx, key).Result()
}

func (r *AppRedis) SetOauthToken(ctx context.Context, key string, token *oauth2.Token) error {
	fmt.Println("access token: ", token.AccessToken)
	fmt.Println("refresh token: ", token.RefreshToken)
	jsonToken, err := json.Marshal(token)
	if err != nil {
		message := "failed to encode token"
		return errors.New(message)
	}
	return r.Redis.Set(ctx, key, jsonToken, time.Duration(TOKEN_MIN)*time.Minute).Err()
}

func (r *AppRedis) GetOauthToken(ctx context.Context, key string) (*oauth2.Token, error) {
	jsonToken, err := r.Redis.Get(ctx, key).Result()
	if err != nil {
		message := "failed to get token"
		return nil, errors.New(message)
	}
	var token oauth2.Token
	err = json.Unmarshal([]byte(jsonToken), &token)
	if err != nil {
		message := "failed to decode token"
		return nil, errors.New(message)
	}

	return &token, nil
}
