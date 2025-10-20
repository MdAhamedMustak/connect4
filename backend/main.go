package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

const (
	ROWS = 6
	COLS = 7
)

type Color string

const (
	Red    Color = "red"
	Yellow Color = "yellow"
	Empty  Color = ""
)

type GameState struct {
	ID            string
	Board         [][]Color
	Player1       *Player
	Player2       *Player
	CurrentPlayer Color
	Winner        string
	StartTime     time.Time
	EndTime       *time.Time
	IsBot         bool
	mutex         sync.RWMutex
}

type Player struct {
	Username     string
	Color        Color
	Conn         *websocket.Conn
	LastSeen     time.Time
	Disconnected bool
}

type Message struct {
	Type          string    `json:"type"`
	Username      string    `json:"username,omitempty"`
	Column        int       `json:"column,omitempty"`
	Board         [][]Color `json:"board,omitempty"`
	CurrentPlayer Color     `json:"current_player,omitempty"`
	Color         Color     `json:"color,omitempty"`
	Opponent      string    `json:"opponent,omitempty"`
	Winner        string    `json:"winner,omitempty"`
	GameID        string    `json:"game_id,omitempty"`
	Message       string    `json:"message,omitempty"`
}

type GameServer struct {
	games          map[string]*GameState
	playerGames    map[string]*GameState  // Track which game each player is in
	waitingPlayers []*Player
	upgrader       websocket.Upgrader
	mutex          sync.RWMutex
	db             *sql.DB
	kafkaWriter    *kafka.Writer
}

type LeaderboardEntry struct {
	Username string `json:"username"`
	Wins     int    `json:"wins"`
}

func NewGameServer(db *sql.DB, kafkaWriter *kafka.Writer) *GameServer {
	return &GameServer{
		games:       make(map[string]*GameState),
		playerGames: make(map[string]*GameState),
		upgrader:    websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		db:          db,
		kafkaWriter: kafkaWriter,
	}
}

func (gs *GameServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := gs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("❌ Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("✓ New WebSocket connection")

	var player *Player
	var game *GameState

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if player != nil && game != nil {
				gs.handleDisconnect(player, game)
			}
			break
		}

		log.Printf("📨 %s from %s", msg.Type, msg.Username)

		switch msg.Type {
		case "join":
			player = &Player{Username: msg.Username, Conn: conn, LastSeen: time.Now()}
			log.Printf("👤 %s joining", player.Username)
			game = gs.matchPlayer(player)
		case "move":
			// Look up the game for this player
			gs.mutex.RLock()
			game = gs.playerGames[player.Username]
			gs.mutex.RUnlock()
			
			if game == nil {
				log.Printf("❌ Move received but no game found for %s", player.Username)
				conn.WriteJSON(Message{Type: "error", Message: "Game not found"})
			} else if player == nil {
				log.Println("❌ Move received but player is nil")
				conn.WriteJSON(Message{Type: "error", Message: "Player not found"})
			} else {
				log.Printf("🎮 %s → column %d", player.Username, msg.Column)
				gs.handleMove(game, player, msg.Column)
			}
		}
	}
}

func (gs *GameServer) matchPlayer(player *Player) *GameState {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	// Check rejoin
	for _, game := range gs.games {
		if game.Winner == "" {
			if game.Player1.Username == player.Username && game.Player1.Disconnected {
				game.Player1.Conn = player.Conn
				game.Player1.Disconnected = false
				player.Color = game.Player1.Color
				gs.sendGameState(game)
				log.Printf("🔄 %s reconnected", player.Username)
				return game
			}
			if game.Player2 != nil && game.Player2.Username == player.Username && game.Player2.Disconnected {
				game.Player2.Conn = player.Conn
				game.Player2.Disconnected = false
				player.Color = game.Player2.Color
				gs.sendGameState(game)
				log.Printf("🔄 %s reconnected", player.Username)
				return game
			}
		}
	}

	// Match with waiting player
	if len(gs.waitingPlayers) > 0 {
		opponent := gs.waitingPlayers[0]
		gs.waitingPlayers = gs.waitingPlayers[1:]
		log.Printf("👥 Matching %s vs %s", opponent.Username, player.Username)
		return gs.createGame(opponent, player, false)
	}

	// Add to waiting list
	gs.waitingPlayers = append(gs.waitingPlayers, player)
	player.Conn.WriteJSON(Message{Type: "waiting"})
	log.Printf("⏳ %s waiting", player.Username)

	// Bot timer
	go func() {
		time.Sleep(10 * time.Second)
		gs.mutex.Lock()
		defer gs.mutex.Unlock()
		for i, p := range gs.waitingPlayers {
			if p == player {
				gs.waitingPlayers = append(gs.waitingPlayers[:i], gs.waitingPlayers[i+1:]...)
				bot := &Player{Username: "Bot", Color: Yellow}
				log.Printf("🤖 Bot joining %s", player.Username)
				gs.createGame(player, bot, true)
				break
			}
		}
	}()

	return nil
}

func (gs *GameServer) createGame(p1, p2 *Player, isBot bool) *GameState {
	gameID := generateID()
	p1.Color = Red
	p2.Color = Yellow

	board := make([][]Color, ROWS)
	for i := range board {
		board[i] = make([]Color, COLS)
	}

	game := &GameState{
		ID: gameID, Board: board, Player1: p1, Player2: p2,
		CurrentPlayer: Red, StartTime: time.Now(), IsBot: isBot,
	}
	gs.games[gameID] = game
	
	// Track which players are in which game
	gs.playerGames[p1.Username] = game
	if !isBot {
		gs.playerGames[p2.Username] = game
	}

	log.Printf("🎮 Game %s: %s vs %s", gameID, p1.Username, p2.Username)

	if p1.Conn != nil {
		p1.Conn.WriteJSON(Message{Type: "game_start", Color: Red, Opponent: p2.Username, CurrentPlayer: Red, GameID: gameID})
	}
	if !isBot && p2.Conn != nil {
		p2.Conn.WriteJSON(Message{Type: "game_start", Color: Yellow, Opponent: p1.Username, CurrentPlayer: Red, GameID: gameID})
	}

	gs.sendKafkaEvent("game_start", map[string]interface{}{
		"game_id": gameID, "player1": p1.Username, "player2": p2.Username, "is_bot": isBot,
	})

	return game
}

func (gs *GameServer) handleMove(game *GameState, player *Player, col int) {
	game.mutex.Lock()
	defer game.mutex.Unlock()

	log.Printf("🎯 %s (%s) → col %d (turn: %s)", player.Username, player.Color, col, game.CurrentPlayer)

	if game.Winner != "" {
		log.Println("❌ Game finished")
		return
	}

	if player.Color != game.CurrentPlayer {
		log.Printf("❌ Not your turn")
		if player.Conn != nil {
			player.Conn.WriteJSON(Message{Type: "error", Message: "Not your turn"})
		}
		return
	}

	if col < 0 || col >= COLS {
		log.Printf("❌ Invalid column")
		return
	}

	row := -1
	for r := ROWS - 1; r >= 0; r-- {
		if game.Board[r][col] == Empty {
			row = r
			break
		}
	}

	if row == -1 {
		log.Printf("❌ Column full")
		if player.Conn != nil {
			player.Conn.WriteJSON(Message{Type: "error", Message: "Column is full"})
		}
		return
	}

	game.Board[row][col] = player.Color
	log.Printf("✓ Placed at [%d,%d]", row, col)

	if gs.checkWinner(game, row, col) {
		game.Winner = string(player.Color)
		endTime := time.Now()
		game.EndTime = &endTime
		log.Printf("🏆 Winner: %s", game.Winner)
		gs.saveGame(game)
		gs.broadcastGameOver(game)
		return
	}

	if gs.isBoardFull(game) {
		game.Winner = "draw"
		endTime := time.Now()
		game.EndTime = &endTime
		log.Println("🤝 Draw")
		gs.saveGame(game)
		gs.broadcastGameOver(game)
		return
	}

	if game.CurrentPlayer == Red {
		game.CurrentPlayer = Yellow
	} else {
		game.CurrentPlayer = Red
	}

	log.Printf("🔄 Turn: %s", game.CurrentPlayer)
	gs.broadcastMove(game)

	if game.IsBot && game.CurrentPlayer == Yellow {
		log.Println("🤖 Bot thinking...")
		go func() {
			time.Sleep(500 * time.Millisecond)
			gs.makeBotMove(game)
		}()
	}
}

func (gs *GameServer) makeBotMove(game *GameState) {
	game.mutex.Lock()
	defer game.mutex.Unlock()

	if game.Winner != "" {
		return
	}

	col := gs.getBotMove(game)
	if col == -1 {
		return
	}

	row := -1
	for r := ROWS - 1; r >= 0; r-- {
		if game.Board[r][col] == Empty {
			row = r
			break
		}
	}

	if row == -1 {
		return
	}

	game.Board[row][col] = Yellow
	log.Printf("🤖 Bot → [%d,%d]", row, col)

	if gs.checkWinner(game, row, col) {
		game.Winner = "yellow"
		endTime := time.Now()
		game.EndTime = &endTime
		log.Println("🤖 Bot wins!")
		gs.saveGame(game)
		gs.broadcastGameOver(game)
		return
	}

	if gs.isBoardFull(game) {
		game.Winner = "draw"
		endTime := time.Now()
		game.EndTime = &endTime
		gs.saveGame(game)
		gs.broadcastGameOver(game)
		return
	}

	game.CurrentPlayer = Red
	gs.broadcastMove(game)
}

func (gs *GameServer) getBotMove(game *GameState) int {
	for col := 0; col < COLS; col++ {
		if gs.canWin(game, col, Yellow) {
			return col
		}
	}
	for col := 0; col < COLS; col++ {
		if gs.canWin(game, col, Red) {
			return col
		}
	}
	if gs.isColumnAvailable(game, 3) {
		return 3
	}
	for _, col := range []int{2, 4, 1, 5, 0, 6} {
		if gs.isColumnAvailable(game, col) {
			return col
		}
	}
	return -1
}

func (gs *GameServer) canWin(game *GameState, col int, color Color) bool {
	row := -1
	for r := ROWS - 1; r >= 0; r-- {
		if game.Board[r][col] == Empty {
			row = r
			break
		}
	}
	if row == -1 {
		return false
	}
	game.Board[row][col] = color
	wins := gs.checkWinner(game, row, col)
	game.Board[row][col] = Empty
	return wins
}

func (gs *GameServer) isColumnAvailable(game *GameState, col int) bool {
	return col >= 0 && col < COLS && game.Board[0][col] == Empty
}

func (gs *GameServer) checkWinner(game *GameState, row, col int) bool {
	color := game.Board[row][col]
	if color == Empty {
		return false
	}
	
	// Check all 4 directions: horizontal, vertical, diagonal-right, diagonal-left
	directions := [][2]int{
		{0, 1},  // horizontal
		{1, 0},  // vertical
		{1, 1},  // diagonal down-right
		{1, -1}, // diagonal down-left
	}

	for _, dir := range directions {
		count := 1 // Count the current disc
		
		// Check positive direction
		for i := 1; i < 4; i++ {
			r := row + dir[0]*i
			c := col + dir[1]*i
			if r < 0 || r >= ROWS || c < 0 || c >= COLS {
				break
			}
			if game.Board[r][c] != color {
				break
			}
			count++
		}
		
		// Check negative direction
		for i := 1; i < 4; i++ {
			r := row - dir[0]*i
			c := col - dir[1]*i
			if r < 0 || r >= ROWS || c < 0 || c >= COLS {
				break
			}
			if game.Board[r][c] != color {
				break
			}
			count++
		}
		
		// If we found 4 or more in a row, we have a winner!
		if count >= 4 {
			log.Printf("🎉 Found %d in a row for %s! (direction: %v)", count, color, dir)
			return true
		}
	}
	
	return false
}

func (gs *GameServer) isBoardFull(game *GameState) bool {
	for _, row := range game.Board {
		for _, cell := range row {
			if cell == Empty {
				return false
			}
		}
	}
	return true
}

func (gs *GameServer) broadcastMove(game *GameState) {
	msg := Message{Type: "move", Board: game.Board, CurrentPlayer: game.CurrentPlayer}
	
	log.Printf("📤 Broadcasting move - Current player: %s", game.CurrentPlayer)
	
	if game.Player1.Conn != nil {
		if err := game.Player1.Conn.WriteJSON(msg); err != nil {
			log.Printf("❌ Error sending to Player1: %v", err)
		} else {
			log.Printf("✓ Sent to %s", game.Player1.Username)
		}
	}
	
	if game.Player2 != nil && game.Player2.Conn != nil {
		if err := game.Player2.Conn.WriteJSON(msg); err != nil {
			log.Printf("❌ Error sending to Player2: %v", err)
		} else {
			log.Printf("✓ Sent to %s", game.Player2.Username)
		}
	}
}

func (gs *GameServer) broadcastGameOver(game *GameState) {
	msg := Message{Type: "game_over", Board: game.Board, Winner: game.Winner}
	
	log.Printf("🏁 Broadcasting game over - Winner: %s", game.Winner)
	log.Printf("   Player1: %s (%s)", game.Player1.Username, game.Player1.Color)
	if game.Player2 != nil {
		log.Printf("   Player2: %s (%s)", game.Player2.Username, game.Player2.Color)
	}
	
	if game.Player1.Conn != nil {
		if err := game.Player1.Conn.WriteJSON(msg); err != nil {
			log.Printf("❌ Error sending game over to Player1: %v", err)
		} else {
			log.Printf("✓ Sent game over to %s", game.Player1.Username)
		}
	}
	
	if game.Player2 != nil && game.Player2.Conn != nil {
		if err := game.Player2.Conn.WriteJSON(msg); err != nil {
			log.Printf("❌ Error sending game over to Player2: %v", err)
		} else {
			log.Printf("✓ Sent game over to %s", game.Player2.Username)
		}
	}
	
	gs.sendKafkaEvent("game_end", map[string]interface{}{
		"game_id": game.ID, "winner": game.Winner, "duration": time.Since(game.StartTime).Seconds(), "is_bot": game.IsBot,
	})
}

func (gs *GameServer) handleDisconnect(player *Player, game *GameState) {
	game.mutex.Lock()
	defer game.mutex.Unlock()
	player.Disconnected = true
	opponent := gs.getOpponent(game, player)
	if opponent != nil && opponent.Conn != nil {
		opponent.Conn.WriteJSON(Message{Type: "opponent_disconnected"})
	}
	go func() {
		time.Sleep(30 * time.Second)
		game.mutex.Lock()
		defer game.mutex.Unlock()
		if player.Disconnected && game.Winner == "" {
			game.Winner = string(gs.getOpponent(game, player).Color)
			endTime := time.Now()
			game.EndTime = &endTime
			gs.saveGame(game)
			if opp := gs.getOpponent(game, player); opp != nil && opp.Conn != nil {
				opp.Conn.WriteJSON(Message{Type: "game_forfeited", Winner: game.Winner})
			}
		}
	}()
}

func (gs *GameServer) getOpponent(game *GameState, player *Player) *Player {
	if game.Player1 == player {
		return game.Player2
	}
	return game.Player1
}

func (gs *GameServer) sendGameState(game *GameState) {
	msg := Message{Type: "move", Board: game.Board, CurrentPlayer: game.CurrentPlayer}
	if game.Player1.Conn != nil && !game.Player1.Disconnected {
		game.Player1.Conn.WriteJSON(msg)
	}
	if game.Player2 != nil && game.Player2.Conn != nil && !game.Player2.Disconnected {
		game.Player2.Conn.WriteJSON(msg)
	}
}

func (gs *GameServer) saveGame(game *GameState) {
	if gs.db == nil {
		log.Println("⚠ Database not available - game not saved")
		return
	}
	
	winner := game.Winner
	if winner == "draw" {
		winner = ""
	}
	
	// Map color names to player usernames
	var winnerUsername string
	if winner == "red" {
		winnerUsername = game.Player1.Username
	} else if winner == "yellow" {
		winnerUsername = game.Player2.Username
	}
	
	_, err := gs.db.Exec(`
		INSERT INTO games (id, player1, player2, winner, start_time, end_time, is_bot) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, game.ID, game.Player1.Username, game.Player2.Username, winnerUsername, game.StartTime, game.EndTime, game.IsBot)
	
	if err != nil {
		log.Println("❌ Error saving game:", err)
	} else {
		log.Printf("✓ Game saved - Winner: %s", winnerUsername)
	}
}

func (gs *GameServer) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if gs.db == nil {
		log.Println("⚠ Leaderboard request but DB not available")
		json.NewEncoder(w).Encode([]LeaderboardEntry{})
		return
	}

	rows, err := gs.db.Query(`
		SELECT winner as username, COUNT(*) as wins 
		FROM games 
		WHERE winner IS NOT NULL AND winner != '' 
		GROUP BY winner 
		ORDER BY wins DESC 
		LIMIT 10
	`)
	
	if err != nil {
		log.Println("❌ Error fetching leaderboard:", err)
		json.NewEncoder(w).Encode([]LeaderboardEntry{})
		return
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.Username, &entry.Wins); err == nil {
			leaderboard = append(leaderboard, entry)
		}
	}
	
	if leaderboard == nil {
		leaderboard = []LeaderboardEntry{}
	}
	
	log.Printf("📊 Leaderboard: %d entries", len(leaderboard))
	json.NewEncoder(w).Encode(leaderboard)
}

func (gs *GameServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy", "version": "1.0.0", "active_games": len(gs.games), "waiting_players": len(gs.waitingPlayers),
	})
	log.Println("✓ Health check")
}

func (gs *GameServer) sendKafkaEvent(eventType string, data map[string]interface{}) {
	if gs.kafkaWriter == nil {
		return
	}
	data["event_type"] = eventType
	jsonData, _ := json.Marshal(data)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	gs.kafkaWriter.WriteMessages(ctx, kafka.Message{Key: []byte(eventType), Value: jsonData})
}

func generateID() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = "abcdefghijklmnopqrstuvwxyz0123456789"[rand.Intn(36)]
	}
	return string(b)
}

func initDB() *sql.DB {
	password := "postgres"
	if p := os.Getenv("DB_PASSWORD"); p != "" {
		password = p
	}
	
	connStr := fmt.Sprintf("host=localhost port=5432 user=postgres password=%s dbname=connect4 sslmode=disable", password)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Println("⚠ Database connection failed:", err)
		log.Println("⚠ Game will work but leaderboard won't be saved")
		return nil
	}
	
	if err := db.Ping(); err != nil {
		log.Println("⚠ Database ping failed:", err)
		log.Println("⚠ Check if PostgreSQL is running and password is correct")
		log.Println("⚠ Game will work but leaderboard won't be saved")
		return nil
	}
	
	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS games (
			id VARCHAR(50) PRIMARY KEY,
			player1 VARCHAR(100) NOT NULL,
			player2 VARCHAR(100) NOT NULL,
			winner VARCHAR(100),
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			is_bot BOOLEAN DEFAULT FALSE
		)
	`)
	if err != nil {
		log.Println("⚠ Error creating table:", err)
	} else {
		log.Println("✓ Database connected & table ready")
	}
	
	return db
}

func initKafka() *kafka.Writer {
	writer := &kafka.Writer{Addr: kafka.TCP("localhost:9092"), Topic: "game-events", Balancer: &kafka.LeastBytes{}}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if writer.WriteMessages(ctx, kafka.Message{Key: []byte("test"), Value: []byte("test")}) != nil {
		log.Println("⚠ Kafka skipped (game works without it)")
		return nil
	}
	log.Println("✓ Kafka connected")
	return writer
}

func main() {
	rand.Seed(time.Now().UnixNano())
	log.Println("🎮 4 in a Row Server")
	log.Println("====================")

	db := initDB()
	if db != nil {
		defer db.Close()
	}

	kafkaWriter := initKafka()
	if kafkaWriter != nil {
		defer kafkaWriter.Close()
	}

	server := NewGameServer(db, kafkaWriter)

	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next(w, r)
		}
	}

	http.HandleFunc("/ws", server.HandleWebSocket)
	http.HandleFunc("/leaderboard", corsMiddleware(server.getLeaderboard))
	http.HandleFunc("/health", corsMiddleware(server.healthCheck))

	log.Println("✓ Server ready on :8080")
	log.Println("📍 http://localhost:8080/health")
	log.Println("📍 ws://localhost:8080/ws")
	log.Println("====================")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("❌ Error:", err)
	}
}