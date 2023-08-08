package cache

const GLOBAL_USAGE_KEY = "global_data"
const BLACKLIST_SET_KEY = "blacklisted_users"

type Cache interface {
	IncrementUsage(userId string, val int) (int64, int64, error)
	FetchUsage(string) (int64, error)
	FetchGlobalUsage() (int64, error)
	GetBlacklistedUsers() ([]string, error)
	BlacklistUser(userId string) error
}
