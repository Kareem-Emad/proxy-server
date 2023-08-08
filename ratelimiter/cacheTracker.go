package ratelimiter

import (
	"context"
	"sproxy/storage/cache"
)

type CacheRateLimiter struct {
	cacheClient cache.Cache
	ctx         context.Context
	// could be safely loaded on restart in memory and maintained since it won't be a huge number( we hope we have a million customer tho :pray: :pray: )
	lockedUsers map[string]bool
	totalLock   bool
}

func (crl *CacheRateLimiter) UpdateTrafficUsageForUser(userId string, requestSize int) (bool, error) {
	if crl.lockedUsers[userId] || crl.totalLock {
		return false, nil
	}

	newUserUsage, newGlobalUsage, err := crl.cacheClient.IncrementUsage(userId, requestSize)
	if err != nil {
		return false, err
	}

	crl.lockedUsers[userId] = newUserUsage >= QUOTA_PER_USER
	if crl.lockedUsers[userId] {
		crl.cacheClient.BlacklistUser(userId)
	}

	crl.totalLock = newGlobalUsage >= GLOBAL_QUOTA_LIMIT
	return true, nil
}

func (crl *CacheRateLimiter) FetchTrafficStatsForUser(userId string) (int64, error) {

	return crl.cacheClient.FetchUsage(userId)

}

func (crl *CacheRateLimiter) FetchGlobalTrafficStats() (int64, error) {
	return crl.cacheClient.FetchGlobalUsage()
}

func NewCacheRateLimiter(ctx context.Context, cacheClient cache.Cache) (CacheRateLimiter, error) {

	if cacheClient == nil {
		cacheClient = cache.NewRedisClient(ctx)
	}

	usage, err := cacheClient.FetchGlobalUsage()
	if err != nil {
		return CacheRateLimiter{}, nil
	}

	users, err := cacheClient.GetBlacklistedUsers()
	if err != nil {
		return CacheRateLimiter{}, err
	}

	lockedUsers := make(map[string]bool)
	for _, usr := range users {
		lockedUsers[usr] = true
	}

	return CacheRateLimiter{
		cacheClient: cacheClient,
		ctx:         ctx,
		lockedUsers: lockedUsers,
		totalLock:   usage >= GLOBAL_QUOTA_LIMIT,
	}, nil
}
