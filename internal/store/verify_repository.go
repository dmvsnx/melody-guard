package store

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type VerifyRepository interface {
	SetVerified(ctx context.Context, guildID, userID string) error
	SetUnverified(ctx context.Context, guildID, userID string) error
	IsVerified(ctx context.Context, guildID, userID string) (bool, error)
	RemoveUserFromGuild(ctx context.Context, guildID, userID string) error
}

type redisVerifyRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) VerifyRepository {
	return &redisVerifyRepository{
		client: client,
	}
}

func (r *redisVerifyRepository) SetVerified(ctx context.Context, guildID, userID string) error {
	return r.client.Set(ctx, "guild:"+guildID+":verified:"+userID, "1", 0).Err()
}

func (r *redisVerifyRepository) SetUnverified(ctx context.Context, guildID, userID string) error {
	return r.client.Del(ctx, "guild:"+guildID+":verified:"+userID).Err()
}

func (r *redisVerifyRepository) IsVerified(ctx context.Context, guildID, userID string) (bool, error) {
	val, err := r.client.Exists(ctx, "guild:"+guildID+":verified:"+userID).Result()
	return val > 0, err
}

func (r *redisVerifyRepository) RemoveUserFromGuild(ctx context.Context, guildID, userID string) error {
	return r.client.Del(ctx, "guild:"+guildID+":verified:"+userID).Err()
}
