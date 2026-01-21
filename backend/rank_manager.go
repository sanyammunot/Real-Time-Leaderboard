package main

import (
	"sync"
)

// RankManager maintains a thread-safe histogram of ratings.
// Since ratings are bounded (100-5000), we use a fixed-size array.
type RankManager struct {
	// histogram stores the count of users for each rating.
	// Index = Rating (0-5000). We only use 100-5000.
	histogram [5001]int32
	mu        sync.RWMutex
}

var GlobalRankManager *RankManager

func InitRankManager() {
	GlobalRankManager = &RankManager{}
}

// GetRank calculates the global rank for a given rating in O(1).
// Rank = (Count of users with rating > given_rating) + 1
func (rm *RankManager) GetRank(rating int) int64 {
	if rating > 5000 {
		return 1
	}
	if rating < 0 {
		rating = 0
	}

	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var rank int64 = 1
	// Sum all users strictly higher than 'rating'
	for r := rating + 1; r <= 5000; r++ {
		rank += int64(rm.histogram[r])
	}
	return rank
}

// UpdateUserRating updates the histogram atomically.
// It decrements the count for the old rating and increments for the new rating.
func (rm *RankManager) UpdateUserRating(oldRating, newRating int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Decrement old rating count if valid
	if oldRating >= 100 && oldRating <= 5000 {
		rm.histogram[oldRating]--
	}

	// Increment new rating count
	if newRating >= 100 && newRating <= 5000 {
		rm.histogram[newRating]++
	}
}

// BulkLoad populates the histogram from an initial dataset (e.g., from Redis/DB).
// Use this on startup.
func (rm *RankManager) BulkLoad(ratings []int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, r := range ratings {
		if r >= 100 && r <= 5000 {
			rm.histogram[r]++
		}
	}
}
