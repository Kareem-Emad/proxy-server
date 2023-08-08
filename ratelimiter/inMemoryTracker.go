package ratelimiter

type InMemoryRateLimiter struct {
	trafficData map[string]int64
	lockedUsers map[string]bool
	totalLock   bool
}

func (imrl *InMemoryRateLimiter) UpdateTrafficUsageForUser(userId string, requestSize int) (bool, error) {
	if imrl.lockedUsers[userId] || imrl.totalLock {
		return false, nil
	}

	imrl.lockedUsers[userId] = imrl.trafficData[userId]+int64(requestSize) >= QUOTA_PER_USER
	imrl.totalLock = imrl.trafficData[GLOBAL_USAGE_KEY]+int64(requestSize) >= GLOBAL_QUOTA_LIMIT

	imrl.trafficData[userId] += int64(requestSize)
	imrl.trafficData[GLOBAL_USAGE_KEY] += int64(requestSize)

	return true, nil
}

func (imrl *InMemoryRateLimiter) FetchTrafficStatsForUser(userId string) (int64, error) {

	return imrl.trafficData[userId], nil
}

func (imrl *InMemoryRateLimiter) FetchGlobalTrafficStats() (int64, error) {

	return imrl.trafficData[GLOBAL_USAGE_KEY], nil
}

func NewInMemoryRateLimiter() InMemoryRateLimiter {

	return InMemoryRateLimiter{
		trafficData: make(map[string]int64),
		lockedUsers: make(map[string]bool),
		totalLock:   false,
	}
}
