package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strconv"
    "sync"

    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/joho/godotenv"
    "github.com/redis/go-redis/v9"
)

// -------------------- GLOBALS -------------------- //

// For Redis:
var ctx = context.Background()
var rdb *redis.Client

// For managing WebSocket connections:
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}
var wsClients = make(map[*websocket.Conn]bool)
var wsMutex sync.Mutex // Protects wsClients

// DeltaRequest is the JSON body for incrementing position
type DeltaRequest struct {
    Delta int `json:"delta"`
}

// PositionResponse is how we broadcast the new position
type PositionResponse struct {
    Position int `json:"position"`
}

func main() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found (this is fine if running in a production environment with real env vars).")
    }

    // 2. Read config from environment
    redisAddr := os.Getenv("REDIS_ADDR")
    redisPass := os.Getenv("REDIS_PASS") 
    redisDBStr := os.Getenv("REDIS_DB")  
    if redisDBStr == "" {
        redisDBStr = "0"
    }
    redisDB, err := strconv.Atoi(redisDBStr)
    if err != nil {
        log.Fatalf("Invalid REDIS_DB value: %v", err)
    }

    // 3. Initialize Redis client using env vars
    rdb = redis.NewClient(&redis.Options{
        Addr:     redisAddr,
        Password: redisPass,
        DB:       redisDB,
    })

    // Test Redis connection
    if err := testRedis(); err != nil {
        log.Fatal("Could not connect to Redis:", err)
    }

    // Setup Gorilla Mux
    r := mux.NewRouter()
    r.Use(corsMiddleware)

    // Routes
    r.HandleFunc("/position", getPosition).Methods("GET", "OPTIONS")
    r.HandleFunc("/position", updatePosition).Methods("POST", "OPTIONS")

    // WebSocket endpoint
    r.HandleFunc("/ws", wsHandler)

    // Read server port from env or default to "8080"
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}

// testRedis pings Redis to confirm connectivity
func testRedis() error {
    _, err := rdb.Ping(ctx).Result()
    return err
}

// -------------------- HANDLERS -------------------- //

// getPosition returns the current position from Redis
func getPosition(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    position, err := rdb.Get(ctx, "carPosition").Int()
    if err == redis.Nil {
        // Key doesn't exist; return 0
        position = 0
    } else if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    _ = json.NewEncoder(w).Encode(PositionResponse{Position: position})
}

// updatePosition increments the position by Delta in Redis, then broadcasts
func updatePosition(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    var req DeltaRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Atomically increment in Redis
    newPos, err := rdb.IncrBy(ctx, "carPosition", int64(req.Delta)).Result()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Clamp if negative
    if newPos < 0 {
        newPos = 0
        _ = rdb.Set(ctx, "carPosition", 0, 0).Err()
    }

    broadcastPosition(int(newPos))

    // Return updated position
    _ = json.NewEncoder(w).Encode(PositionResponse{Position: int(newPos)})
}

// wsHandler upgrades the connection to a WebSocket and adds it to our clients
func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Add this connection to our set of clients
    wsMutex.Lock()
    wsClients[conn] = true
    wsMutex.Unlock()

    log.Println("New WebSocket client connected")

    // Optionally send them the current position
    go sendCurrentPosition(conn)

    // Read loop (we ignore actual messages)
    go handleWSRead(conn)
}

// handleWSRead keeps reading in case the client wants to close or send data
func handleWSRead(conn *websocket.Conn) {
    defer func() {
        wsMutex.Lock()
        delete(wsClients, conn)
        wsMutex.Unlock()
        conn.Close()
        log.Println("WebSocket client disconnected")
    }()

    for {
        if _, _, err := conn.NextReader(); err != nil {
            break
        }
    }
}

// broadcastPosition sends the given `pos` to all connected WebSocket clients.
func broadcastPosition(pos int) {
    msg, _ := json.Marshal(PositionResponse{Position: pos})

    wsMutex.Lock()
    defer wsMutex.Unlock()

    for clientConn := range wsClients {
        err := clientConn.WriteMessage(websocket.TextMessage, msg)
        if err != nil {
            log.Println("Error writing to WebSocket client:", err)
            clientConn.Close()
            delete(wsClients, clientConn)
        }
    }
}

// sendCurrentPosition fetches the current position from Redis and sends it to a single WebSocket connection.
func sendCurrentPosition(conn *websocket.Conn) {
    position, err := rdb.Get(ctx, "carPosition").Int()
    if err == redis.Nil {
        position = 0
    } else if err != nil {
        log.Println("Error reading position:", err)
        return
    }

    msg, _ := json.Marshal(PositionResponse{Position: position})
    if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
        log.Println("Error sending current position to new client:", err)
    }
}

// -------------------- MIDDLEWARE -------------------- //
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        w.Header().Set("Access-Control-Max-Age", "3600")

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        next.ServeHTTP(w, r)
    })
}
