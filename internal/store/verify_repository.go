package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type VerifyRepository interface {
	SetVerified(ctx context.Context, guildID, userID string) error
	SetUnverified(ctx context.Context, guildID, userID string) error
	IsVerified(ctx context.Context, guildID, userID string) (bool, error)
	RemoveUserFromGuild(ctx context.Context, guildID, userID string) error
	SetJoinTime(ctx context.Context, guildID, userID string, timestamp time.Time) error
	GetJoinTime(ctx context.Context, guildID, userID string) (*time.Time, error)
	GetAllUnverifiedKeys(ctx context.Context) ([]string, error)
	RemoveJoinTime(ctx context.Context, guildID, userID string) error
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

func (r *redisVerifyRepository) SetJoinTime(ctx context.Context, guildID, userID string, timestamp time.Time) error {
	return r.client.Set(ctx, "guild:"+guildID+":unverified:"+userID, timestamp.Unix(), 0).Err()
}

func (r *redisVerifyRepository) GetJoinTime(ctx context.Context, guildID, userID string) (*time.Time, error) {
	val, err := r.client.Get(ctx, "guild:"+guildID+":unverified:"+userID).Int64()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t := time.Unix(val, 0)
	return &t, nil
}

func (r *redisVerifyRepository) GetAllUnverifiedKeys(ctx context.Context) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := r.client.Scan(ctx, cursor, "guild:*:unverified:*", 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if next == 0 {
			break
		}
		cursor = next
	}
	return keys, nil
}

func (r *redisVerifyRepository) RemoveJoinTime(ctx context.Context, guildID, userID string) error {
	return r.client.Del(ctx, "guild:"+guildID+":unverified:"+userID).Err()
}
