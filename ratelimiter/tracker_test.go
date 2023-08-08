package ratelimiter

import (
	"context"
	"sproxy/ratelimiter/mocks"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func getNewRateLimiter(rateLimiterInUse string) RateLimiter {
	if rateLimiterInUse == "cache" {
		rdb := redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
		})
		rdb.FlushAll(context.Background())

		rl, err := NewCacheRateLimiter(context.Background(), nil)
		if err != nil {
			panic(err)
		}
		return &rl
	}

	rl := NewInMemoryRateLimiter()
	return &rl
}

// INTEGRATION
func TestTrafficUpdateFlowIntegration(t *testing.T) {

	rateLimiters := []string{"cache", "inMemory"}

	for _, rl := range rateLimiters {
		t.Run("should reflect updated quota for user when fetching his stats and global stats", func(subT *testing.T) {
			testTracker := getNewRateLimiter(rl)

			userId := "testID"
			userId2 := "testID2"
			updateSize := 200
			updateSize2 := 400
			isUpdated, _ := testTracker.UpdateTrafficUsageForUser(userId, updateSize)

			globalUsage, _ := testTracker.FetchGlobalTrafficStats()
			assert.Equal(subT, int64(updateSize), globalUsage, "global usage should be equal to current user usage")

			isUpdated2, _ := testTracker.UpdateTrafficUsageForUser(userId2, updateSize2)

			assert.Equal(subT, true, isUpdated, "quota should be updated succesfully since we did not pass max quota")
			assert.Equal(subT, true, isUpdated2, "quota should be updated succesfully since we did not pass max quota")

			currentUsage, _ := testTracker.FetchTrafficStatsForUser(userId)
			currentUsage2, _ := testTracker.FetchTrafficStatsForUser(userId2)

			assert.Equal(subT, int64(updateSize), currentUsage, "current usage should be equal to added value")
			assert.Equal(subT, int64(updateSize2), currentUsage2, "current usage should be equal to added value")

			globalUsage, _ = testTracker.FetchGlobalTrafficStats()
			assert.Equal(subT, int64(updateSize)+int64(updateSize2), globalUsage, "global usage should be the sum of both users' usages")
		})

		t.Run("should stop updating user quota if user limit is reached", func(subT *testing.T) {
			testTracker := NewInMemoryRateLimiter()

			userId := "testIDx"

			updated, _ := testTracker.UpdateTrafficUsageForUser(userId, QUOTA_PER_USER)
			assert.Equal(subT, true, updated, "should allow updating quota since request size is tightly equal to user total quota")

			updated, _ = testTracker.UpdateTrafficUsageForUser(userId, 1)
			assert.Equal(subT, false, updated, "should not allow update since max quota was already reached")
		})

		t.Run("should stop updating user quota if global limit is reached", func(subT *testing.T) {
			testTracker := NewInMemoryRateLimiter()

			updated, _ := testTracker.UpdateTrafficUsageForUser("U1", QUOTA_PER_USER)
			assert.Equal(subT, true, updated, "should allow updating quota since GLOBAL quota limit is not passed yet")

			updated, _ = testTracker.UpdateTrafficUsageForUser("U2", QUOTA_PER_USER)
			assert.Equal(subT, true, updated, "should allow updating quota since since GLOBAL quota limit is not passed yet")

			updated, _ = testTracker.UpdateTrafficUsageForUser("U3", 1*1024)
			assert.Equal(subT, true, updated, "should allow updating quota since request since current total usage is equal to global quota")

			updated, _ = testTracker.UpdateTrafficUsageForUser("U4", 1*1024)
			assert.Equal(subT, false, updated, "should not allow updating quota since request since current total usage is over  global quota")
		})

	}
}

func TestTrafficUpdateFlowForCache(t *testing.T) {

	t.Run("should accept update for user if it's within user quota", func(subT *testing.T) {
		mockedCache := mocks.NewCache(subT)
		mockedCache.On("FetchGlobalUsage").Return(func() (int64, error) {
			return 0, nil
		})
		mockedCache.On("GetBlacklistedUsers", mock.Anything).Return(func() ([]string, error) {
			return []string{}, nil
		})
		testTracker, _ := NewCacheRateLimiter(context.Background(), mockedCache)

		mockedCache.On("IncrementUsage", mock.Anything, mock.Anything).Return(func(userId string, val int) (int64, int64, error) {
			return 100, 200, nil
		})

		updated, _ := testTracker.UpdateTrafficUsageForUser("u1", 200)
		assert.Equal(subT, true, updated, "should allow updating quota since user limit is not passed yet")

	})

	t.Run("should reject update for user if it's over user quota", func(subT *testing.T) {
		mockedCache := mocks.NewCache(subT)
		mockedCache.On("BlacklistUser", mock.Anything).Return(func(userId string) error {
			return nil
		})
		mockedCache.On("GetBlacklistedUsers", mock.Anything).Return(func() ([]string, error) {
			return []string{}, nil
		})

		mockedCache.On("FetchGlobalUsage").Return(func() (int64, error) {
			return 0, nil
		})
		testTracker, _ := NewCacheRateLimiter(context.Background(), mockedCache)

		mockedCache.On("IncrementUsage", mock.Anything, mock.Anything).Return(func(userId string, val int) (int64, int64, error) {
			v1 := int64(100)
			v2 := int64(QUOTA_PER_USER + 1)

			return v2, v1, nil
		})

		updated, _ := testTracker.UpdateTrafficUsageForUser("u1", 200)
		assert.Equal(subT, true, updated, "should allow updating quota since user limit is passed first time")

		updated, _ = testTracker.UpdateTrafficUsageForUser("u1", 200)
		assert.Equal(subT, false, updated, "should not allow updating quota since user limit is passed")
	})

	t.Run("should reject update for user if it's over global quota", func(subT *testing.T) {
		mockedCache := mocks.NewCache(subT)

		mockedCache.On("IncrementUsage", mock.Anything, mock.Anything).Return(func(userId string, val int) (int64, int64, error) {
			v1 := int64(GLOBAL_QUOTA_LIMIT + 1)
			v2 := int64(100)

			return v2, v1, nil
		})
		mockedCache.On("GetBlacklistedUsers", mock.Anything).Return(func() ([]string, error) {
			return []string{}, nil
		})

		mockedCache.On("FetchGlobalUsage").Return(func() (int64, error) {
			return 0, nil
		})

		testTracker, _ := NewCacheRateLimiter(context.Background(), mockedCache)

		updated, _ := testTracker.UpdateTrafficUsageForUser("u1", 200)
		assert.Equal(subT, true, updated, "should allow updating quota since global limit is passed first time")

		updated, _ = testTracker.UpdateTrafficUsageForUser("u1", 200)
		assert.Equal(subT, false, updated, "should not allow updating quota since global limit is passed")
	})

}
