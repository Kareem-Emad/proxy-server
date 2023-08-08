package ratelimiter

const QUOTA_PER_USER = 2 * 1024
const GLOBAL_QUOTA_LIMIT = 5 * 1024
const GLOBAL_USAGE_KEY = "global_data"

type RateLimiter interface {
	UpdateTrafficUsageForUser(userId string, requestSize int) (bool, error)
	FetchTrafficStatsForUser(userId string) (int64, error)
	FetchGlobalTrafficStats() (int64, error)
}
