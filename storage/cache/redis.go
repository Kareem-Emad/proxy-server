package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	cacheClient *redis.Client
	ctx         context.Context
}

func (rc RedisCache) IncrementUsage(userId string, val int) (int64, int64, error) {
	newUserUsage, err := rc.cacheClient.IncrBy(rc.ctx, userId, int64(val)).Result()
	if err != nil {
		return 0, 0, err
	}

	newGlobalUsage, err := rc.cacheClient.IncrBy(rc.ctx, GLOBAL_USAGE_KEY, int64(val)).Result()

	return newUserUsage, newGlobalUsage, err
}

func (rc RedisCache) FetchGlobalUsage() (int64, error) {
	val, err := rc.cacheClient.Get(rc.ctx, GLOBAL_USAGE_KEY).Int64()

	if err != redis.Nil {
		return val, err
	}

	return val, nil
}

func (rc RedisCache) FetchUsage(userId string) (int64, error) {
	val, err := rc.cacheClient.Get(rc.ctx, userId).Int64()

	if err != redis.Nil {
		return val, err
	}

	return val, nil
}

func (rc RedisCache) GetBlacklistedUsers() ([]string, error) {
	return rc.cacheClient.SMembers(rc.ctx, BLACKLIST_SET_KEY).Result()
}

func (rc RedisCache) BlacklistUser(userId string) error {
	_, err := rc.cacheClient.SAdd(rc.ctx, userId).Result()
	return err
}

func NewRedisClient(ctx context.Context) RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return RedisCache{
		ctx:         ctx,
		cacheClient: rdb,
	}
}
