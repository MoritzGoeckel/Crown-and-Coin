package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"crown_and_coin/engine"
	"crown_and_coin/jsonapi"

	"github.com/gorilla/websocket"
)

type User struct {
	Name   string
	Secret string
}

type ClientConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *ClientConn) send(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *ClientConn) sendError(message string) {
	resp, _ := json.Marshal(ErrorResponse{Success: false, Error: message})
	c.send(resp)
}

type Server struct {
	users    map[string]*User // name -> user
	mu       sync.RWMutex
	api      *jsonapi.GameAPI
	upgrader websocket.Upgrader

	clients   map[string]*ClientConn
	clientsMu sync.RWMutex
}

type ClientMessage struct {
	User    string          `json:"user"`
	Secret  string          `json:"secret"`
	Payload json.RawMessage `json:"payload"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func NewServer() *Server {
	return &Server{
		users:   make(map[string]*User),
		clients: make(map[string]*ClientConn),
		api:     jsonapi.NewGameAPIWithDice(engine.NewSeededDice(42)),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func generateSecret() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *Server) registerUser(name, secret string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; exists {
		return fmt.Errorf("user already exists")
	}

	s.users[name] = &User{Name: name, Secret: secret}
	return nil
}

func (s *Server) authenticateUser(name, secret string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[name]
	if !exists {
		return false
	}
	return user.Secret == secret
}

func (s *Server) isAdmin(name string) bool {
	return name == "admin"
}

func (s *Server) canSendMessage(user string, payload json.RawMessage) bool {
	if s.isAdmin(user) {
		return true
	}

	var msg struct {
		Type     string `json:"type"`
		PlayerID string `json:"player_id"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return false
	}

	switch msg.Type {
	case "get_state", "get_players", "get_connected_players":
		return true
	case "get_actions", "get_queued":
		return msg.PlayerID == user
	case "submit":
		// Check if all actions belong to this user
		var submitMsg struct {
			Actions []struct {
				PlayerID string `json:"player_id"`
			} `json:"actions"`
		}
		if err := json.Unmarshal(payload, &submitMsg); err != nil {
			return false
		}
		for _, action := range submitMsg.Actions {
			if action.PlayerID != user {
				return false
			}
		}
		return true
	case "add_country", "add_merchant", "remove_merchant", "advance":
		return false // admin only
	default:
		return false
	}
}

func (s *Server) getConnectedPlayerNames() []string {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	names := make([]string, 0, len(s.clients))
	for name := range s.clients {
		if name != "admin" {
			names = append(names, name)
		}
	}
	return names
}

func (s *Server) broadcastConnectedPlayers() {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	names := make([]string, 0, len(s.clients))
	for name := range s.clients {
		if name != "admin" {
			names = append(names, name)
		}
	}

	msg, _ := json.Marshal(map[string]interface{}{
		"type":    "connected_players",
		"success": true,
		"players": names,
	})

	for _, client := range s.clients {
		client.send(msg)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WebSocket connection from %s", r.RemoteAddr)

	client := &ClientConn{conn: conn}
	var connUser string

	defer func() {
		if connUser != "" {
			s.clientsMu.Lock()
			if existing, ok := s.clients[connUser]; ok && existing == client {
				delete(s.clients, connUser)
			}
			s.clientsMu.Unlock()
			log.Printf("Player disconnected: %s", connUser)
			s.broadcastConnectedPlayers()
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Printf("Invalid message format: %v", err)
			client.sendError("invalid message format")
			continue
		}

		if !s.authenticateUser(clientMsg.User, clientMsg.Secret) {
			log.Printf("Authentication failed for user: %s", clientMsg.User)
			client.sendError("authentication failed")
			continue
		}

		// Register connection on first authenticated message
		if connUser == "" {
			connUser = clientMsg.User
			s.clientsMu.Lock()
			s.clients[connUser] = client
			s.clientsMu.Unlock()
			log.Printf("Player connected: %s", connUser)
			s.broadcastConnectedPlayers()
		}

		if !s.canSendMessage(clientMsg.User, clientMsg.Payload) {
			log.Printf("Permission denied for user %s", clientMsg.User)
			client.sendError("permission denied")
			continue
		}

		// Handle get_connected_players separately (not part of game engine)
		var msgType struct {
			Type string `json:"type"`
		}
		json.Unmarshal(clientMsg.Payload, &msgType)

		if msgType.Type == "get_connected_players" {
			names := s.getConnectedPlayerNames()
			resp, _ := json.Marshal(map[string]interface{}{
				"type":    "connected_players",
				"success": true,
				"players": names,
			})
			client.send(resp)
			continue
		}

		response, err := s.api.ProcessMessage(clientMsg.Payload)
		if err != nil {
			log.Printf("Engine error: %v", err)
			client.sendError(err.Error())
			continue
		}

		if err := client.send(response); err != nil {
			log.Printf("Write error: %v", err)
			break
		}
	}
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name   string `json:"name"`
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Secret == "" {
		http.Error(w, "Name and secret required", http.StatusBadRequest)
		return
	}

	if req.Name == "admin" {
		http.Error(w, "Cannot register as admin", http.StatusForbidden)
		return
	}

	if err := s.registerUser(req.Name, req.Secret); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	log.Printf("User registered: %s", req.Name)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name   string `json:"name"`
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Secret == "" {
		http.Error(w, "Name and secret required", http.StatusBadRequest)
		return
	}

	if !s.authenticateUser(req.Name, req.Secret) {
		http.Error(w, "Invalid username or secret", http.StatusUnauthorized)
		return
	}

	log.Printf("User logged in: %s", req.Name)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func main() {
	server := NewServer()

	// Create admin user with generated secret
	adminSecret := generateSecret()
	server.users["admin"] = &User{Name: "admin", Secret: adminSecret}
	fmt.Printf("Admin secret: %s\n", adminSecret)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// API endpoints
	http.HandleFunc("/register", server.handleRegister)
	http.HandleFunc("/login", server.handleLogin)
	http.HandleFunc("/ws", server.handleWebSocket)

	addr := ":8080"
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
