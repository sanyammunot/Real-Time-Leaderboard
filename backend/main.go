package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ctx         = context.Background()
	redisClient *redis.Client
	db          *pgxpool.Pool
)

func main() {
	// 1. Initialize Redis
	redisURL := os.Getenv("REDIS_URL")
	var redisOpts *redis.Options
	var err error

	if redisURL != "" {
		redisOpts, err = redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("Warning: Failed to parse REDIS_URL: %v", err)
		}
	}

	if redisOpts == nil {
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}
		redisOpts = &redis.Options{
			Addr: redisHost + ":6379",
		}
	}

	redisClient = redis.NewClient(redisOpts)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		fmt.Println("Connected to Redis")
	}

	// 3. Connect to Postgres (using Pool)
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		pgHost := os.Getenv("POSTGRES_HOST")
		if pgHost == "" {
			pgHost = "localhost"
		}
		connString = fmt.Sprintf("postgres://admin:password@%s:5432/leaderboard?sslmode=disable", pgHost)
	}
	// Create a connection pool instead of a single connection
	config, err = pgxpool.ParseConfig(connString)
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}

	// Adjust pool settings if needed (optional)
	config.MaxConns = 50 // Limit max connections

	db, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		// Retry logic or just log fatal for MVP
		log.Printf("Failed to connect to Postgres (waiting for container?): %v", err)
	} else {
		fmt.Println("Connected to Postgres")
		defer db.Close()

		// Create Table
		_, err = db.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				username VARCHAR(255) UNIQUE NOT NULL,
				rating INT NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_users_username_trgm ON users USING btree (username varchar_pattern_ops);
			CREATE INDEX IF NOT EXISTS idx_users_rating ON users (rating DESC);
		`)
		if err != nil {
			log.Printf("Failed to create table: %v", err)
		}
	}

	// 4. Initialize RankManager
	InitRankManager()

	// 5. Seed Data (Async)
	go func() {
		if db == nil {
			return
		}

		// Check if data exists
		var count int
		db.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count)
		if count > 0 {
			fmt.Printf("Database has %d users, loading RankManager...\n", count)
			// Load Histogram from DB
			rows, _ := db.Query(ctx, "SELECT rating FROM users")
			for rows.Next() {
				var r int
				rows.Scan(&r)
				GlobalRankManager.UpdateUserRating(-1, r) // just increment
			}
			fmt.Println("RankManager warmed up from DB.")
			return
		}

		fmt.Println("Seeding 10k users...")
		SeedUsers()
		fmt.Println("Seeding complete. RankManager warmed up.")
	}()

	// 6. Define Routes
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.GET("/leaderboard", GetLeaderboard)
	r.GET("/search", SearchUser)
	r.POST("/simulate", SimulateScoreUpdate)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server running on port %s\n", port)
	r.Run(":" + port)
}

// --- Handlers & Helpers ---

func SeedUsers() {
	// Generate 10k users
	batchSize := 1000
	usernames := []string{}
	ratings := []int{}

	for i := 0; i < 10000; i++ {
		u := fmt.Sprintf("user_%d", i)
		r := 100 + (i % 4900) // spread 100-5000
		usernames = append(usernames, u)
		ratings = append(ratings, r)

		if len(usernames) >= batchSize {
			insertBatch(usernames, ratings)
			usernames = []string{}
			ratings = []int{}
		}

		// Update Histogram
		GlobalRankManager.UpdateUserRating(-1, r)
	}
	if len(usernames) > 0 {
		insertBatch(usernames, ratings)
	}
}

func insertBatch(users []string, ratings []int) {
	// Simple batch insert
	// In real pgx, use CopyFrom for speed, but individual inserts/multi-value insert is fine for 10k
	// Construct big INSERT statement
	valStr := ""
	args := []interface{}{}
	argId := 1
	for i := range users {
		if i > 0 {
			valStr += ","
		}
		valStr += fmt.Sprintf("($%d, $%d)", argId, argId+1)
		args = append(args, users[i], ratings[i])
		argId += 2
	}

	query := "INSERT INTO users (username, rating) VALUES " + valStr + " ON CONFLICT DO NOTHING"
	_, err := db.Exec(ctx, query, args...)
	if err != nil {
		log.Printf("Batch insert failed: %v", err)
	}
}

func GetLeaderboard(c *gin.Context) {
	// Fetch top 50 users from Postgres
	// Logic: DB gives us the ORDER. RankManager gives us the RANK.
	rows, err := db.Query(ctx, "SELECT username, rating FROM users ORDER BY rating DESC LIMIT 50")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type UserResp struct {
		Rank     int64  `json:"rank"`
		Username string `json:"username"`
		Rating   int    `json:"rating"`
	}
	var resp []UserResp

	for rows.Next() {
		var u string
		var r int
		rows.Scan(&u, &r)

		// Enriched Rank (Live from memory)
		rank := GlobalRankManager.GetRank(r)

		resp = append(resp, UserResp{
			Rank:     rank,
			Username: u,
			Rating:   r,
		})
	}

	c.JSON(200, resp)
}

func SearchUser(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{"error": "q param required"})
		return
	}

	sql := "SELECT username, rating FROM users WHERE username ILIKE $1 ORDER BY length(username), username LIMIT 50"
	rows, err := db.Query(ctx, sql, query+"%")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type UserResp struct {
		Rank     int64  `json:"rank"`
		Username string `json:"username"`
		Rating   int    `json:"rating"`
	}
	var resp []UserResp

	for rows.Next() {
		var u string
		var r int
		rows.Scan(&u, &r)
		rank := GlobalRankManager.GetRank(r)
		resp = append(resp, UserResp{Rank: rank, Username: u, Rating: r})
	}
	c.JSON(200, resp)
}

func SimulateScoreUpdate(c *gin.Context) {
	var body struct {
		Username  string `json:"username"`
		NewRating int    `json:"new_rating"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Start Transaction
	tx, err := db.Begin(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	// 1. Get Old Rating with LOCK
	var oldRating int
	err = tx.QueryRow(ctx, "SELECT rating FROM users WHERE username=$1 FOR UPDATE", body.Username).Scan(&oldRating)
	if err != nil {
		// If user not found, we can't update. Rolling back automatically via defer.
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	// 2. Update DB
	_, err = tx.Exec(ctx, "UPDATE users SET rating=$1 WHERE username=$2", body.NewRating, body.Username)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 3. Commit Transaction
	if err := tx.Commit(ctx); err != nil {
		c.JSON(500, gin.H{"error": "failed to commit transaction"})
		return
	}

	// 4. Update RankManager (Live) - Only after successful commit
	GlobalRankManager.UpdateUserRating(oldRating, body.NewRating)

	c.JSON(200, gin.H{"status": "updated", "old": oldRating, "new": body.NewRating})
}
