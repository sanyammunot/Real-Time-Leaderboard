package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Config
const (
	TargetURL    = "http://localhost:8080"
	NumUsers     = 1000 // Number of concurrent "players" updating ratings
	UpdateRate   = 50   // ms between updates per user
	TestDuration = 3 * time.Minute
)

func main() {
	fmt.Printf("Starting Load Test: %d users updating every %dms for %s\n", NumUsers, UpdateRate, TestDuration)

	var wg sync.WaitGroup
	wg.Add(NumUsers)

	start := time.Now()
	requestCount := 0
	var mu sync.Mutex

	// Monitor Routine: Checks rank of ALL 1000 users every minute
	go func() {
		// Initial wait before first check
		time.Sleep(10 * time.Second)

		chkTicker := time.NewTicker(60 * time.Second)
		defer chkTicker.Stop()

		for range chkTicker.C {
			fmt.Printf("\n[Monitor] Conducting periodic check of %d users...\n", NumUsers)
			startCheck := time.Now()

			// Limit check concurrency to avoid self-DDoS during the check, or just run sequential
			// Sequential check of 1000 users might take 1-2 seconds.
			for i := 0; i < NumUsers; i++ {
				checkUserRank(fmt.Sprintf("user_%d", i))
			}
			fmt.Printf("[Monitor] Check complete in %.2fs\n", time.Since(startCheck).Seconds())
		}
	}()

	// Load Generators
	for i := 0; i < NumUsers; i++ {
		go func(id int) {
			defer wg.Done()

			username := fmt.Sprintf("user_%d", id)
			ticker := time.NewTicker(time.Duration(UpdateRate) * time.Millisecond)
			defer ticker.Stop()

			timeout := time.After(TestDuration)

			for {
				select {
				case <-timeout:
					return
				case <-ticker.C:
					// Random new rating between 1000 and 4000
					newRating := rand.Intn(3000) + 1000
					updateRating(username, newRating)

					mu.Lock()
					requestCount++
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start).Seconds()
	fmt.Printf("\nTest Complete!\nTotal Requests: %d\nTPS (Transactions Per Second): %.2f\n", requestCount, float64(requestCount)/duration)
}

func updateRating(username string, rating int) {
	payload := map[string]interface{}{
		"username":   username,
		"new_rating": rating,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(TargetURL+"/simulate", "application/json", bytes.NewBuffer(body))
	if err != nil {
		// keeping silent on errors for speed, but detailed logs would go here
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
}

func checkUserRank(username string) {
	resp, err := http.Get(TargetURL + "/search?q=" + username)
	if err != nil {
		fmt.Println("Error searching user:", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	// We expect a JSON array, find the rank in it simply for display
	fmt.Printf("[Live Check] %s: %s\n", username, string(bodyBytes))
}
