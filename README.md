Go WebSocket + Redis Example
Overview
This project demonstrates how to build a real-time web application in Go using:

Gorilla Mux for HTTP routing
Gorilla WebSocket for real-time, bi-directional communication
Redis for storing and updating shared state atomically (in this case, a “car” position)
Basic CORS middleware to allow cross-origin requests (useful for connecting from a React frontend)
When the client presses a Forward or Backward button (in a separate React application), this Go server updates the position in Redis. It then broadcasts the new position via WebSocket to all connected clients, so everyone sees the changes instantly.

Learning Outcomes (Go-Focused)
HTTP Routing with Gorilla Mux

Define routes like GET /position and POST /position.
Restrict handler functions to specific methods (e.g., .Methods("GET", "OPTIONS")).
Serve a WebSocket upgrade endpoint at /ws.
JSON Handling

Use json.NewEncoder(w).Encode(...) and json.NewDecoder(r.Body).Decode(...) to serialize and deserialize JSON.
Maintain simple request/response structs (DeltaRequest, PositionResponse) for clarity.
Redis Integration

Connect to Redis using the go-redis client.
Perform atomic increments (IncrBy) to avoid data races on shared state.
Use a context.Context to enable potential timeouts or cancellations.
Environment Variables

Read configuration (Redis address, password, DB index, server port) from environment variables.
This allows flexible setup across different environments (local, staging, production).
Example usage: os.Getenv("REDIS_ADDR"), os.Getenv("REDIS_PASS"), os.Getenv("PORT"), etc.
You might set these variables using shell commands like export REDIS_ADDR=... or rely on your hosting platform’s environment configuration.
WebSockets (Gorilla WebSocket)

Convert an HTTP connection to a WebSocket with websocket.Upgrader.
Maintain a global set of client connections (wsClients) using a sync.Mutex to guard concurrent access.
Broadcast messages to all connected sockets whenever the position updates.
Detect disconnections with a read loop (conn.NextReader()) and remove the client from the set.
Concurrency and Synchronization

Use goroutines to manage each WebSocket client’s read loop.
Protect shared maps (wsClients) with a mutex to avoid race conditions.
CORS Middleware

Example of a simple middleware that sets Access-Control-Allow-Origin, Access-Control-Allow-Methods, etc.
Essential for allowing browser clients from different domains to call this API.
How to Run
Set Environment Variables (via your shell or hosting platform):

bash
Copy
Edit
export REDIS_ADDR="localhost:6379"
export REDIS_PASS=""
export REDIS_DB="0"
export PORT="8080"
Adjust values according to your Redis setup and desired server port.

Install Dependencies

bash
Copy
Edit
go mod tidy
Run the Server

bash
Copy
Edit
go run main.go
If everything is configured correctly, you’ll see:

nginx
Copy
Edit
Server starting on port 8080
Connect a Frontend

A React (or any other) frontend can fetch GET /position for the initial position,
POST {"delta": 50} or {"delta": -50} to /position to move forward/backward,
and subscribe to ws://localhost:8080/ws for real-time updates.
Example Architecture
Frontend (React/JS)

Sends POST /position with {"delta": 50} to move forward.
Opens a WebSocket to ws://localhost:8080/ws to receive position updates instantly.
Backend (This Go App)

Updates the position in Redis using INCRBY, ensuring concurrency safety.
Broadcasts PositionResponse{Position: newPos} to all active WebSocket clients.
Redis

Stores the shared position.
Handles atomic increments so multiple users cannot overwrite each other’s updates.
