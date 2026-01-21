# Real-Time Leaderboard System

This is an ultra-high-performance, real-time leaderboard system designed to handle **millions of concurrent users** with immediate rank updates. It uses a hybrid **In-Memory + Persistent Database** architecture to achieve O(1) rank retrieval while ensuring data durability.

## Features
- **Real-Time Ranking**: Users get their new rank immediately after a score update.
- **High Concurrency**: Handles thousands of updates per second (TPS) with atomic consistency.
- **Hybrid Architecture**:
    - **PostgreSQL**: Source of truth for data durability.
    - **In-Memory Histogram**: Source of truth for live ranking (milliseconds latency).
- **Search**: Efficient prefix search for user lookup.
- **Scalable**: Built with Go, optimized for vertical and horizontal scaling.
- **Cross-Platform Frontend**: React Native (Expo) app for Web, Android, and iOS.

---

## 🛠 Technology Stack

### Backend
- **Language**: Golang (1.23+)
- **Framework**: Gin Gonic (HTTP Web Framework)
- **Database**: PostgreSQL 15
- **Cache**: Redis 7 
- **Key Libraries**: `pgx` (Driver), `go-redis`

### Frontend
- **Framework**: React Native (Expo)
- **Platforms**: Web, Android
- **Networking**: `fetch` API

### Infrastructure
- **Containerization**: Docker & Docker Compose
- **Hosting**:
    - **Backend**: Render.com (Docker)
    - **Frontend**: Vercel 

---

## 🏗 Architecture

### The Problem
Traditional leaderboards use `ORDER BY score DESC`.
- For 1 Million users, sorting takes $O(N \log N)$ time.
- Running this query on every page view crashes the database.

### The Solution: Frequency Arrays (Buckets)
Since game scores usually have a fixed range (e.g., 100 to 5000), we don't need to sort users. We count them.

1.  **Memory Structure**: An array `[5001]int` (We only use index 100-5000).
    -   Index `i` = Score.
    -   Value `arr[i]` = Number of users having that score.
2.  **Get Rank ($O(1)$)**:
    -   Sum all users in buckets *higher* than my score.
    -   Use a Fenwick Tree (Binary Indexed Tree) for $O(\log K)$ or simple suffix sum for small ranges.
3.  **Update ($O(1)$)**:
    -   Decrement count of old score.
    -   Increment count of new score.

### Data Flow
1.  **Request**: `POST /simulate` { "username": "player1", "new_score": 2500 }
2.  **Transaction Start**:
    -   `SELECT ... FOR UPDATE` locks the user row in Postgres.
3.  **Update DB**: Write new score to disk.
4.  **Update Memory**: Atomically update the in-memory Histogram.
5.  **Commit**: Release lock.

---

## 📚 API Endpoints

### 1. Get Leaderboard
Returns the top 50 users (sorted by score) with their live ranks.
- **GET** `/leaderboard`
- **Response**:
    ```json
    [
        { "rank": 1, "username": "user_123", "rating": 4500 },
        { "rank": 2, "username": "user_99", "rating": 4480 }
    ]
    ```

### 2. Search User
Find a specific user by username prefix.
- **GET** `/search?q=sanyam`

### 3. Simulate Score (Game Server)
Simulates a game reporting a new score. Atomic & Thread-safe.
- **POST** `/simulate`
- **Body**:
    ```json
    { "username": "user_123", "new_rating": 3050 }
    ```

---

## 🧪 Load Testing
We include a high-performance load tester written in Go to verify stability.

```bash
cd loadtester
go run main.go
```
- Spawns **1000 concurrent users**.
- Sends updates every **50ms**.
- Monitors real-time rank accuracy.

---

**Author**: Sanyam Munot
