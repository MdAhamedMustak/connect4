package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type GameEvent struct {
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"-"`
}

type Analytics struct {
	reader       *kafka.Reader
	gamesStarted int
	gamesEnded   int
	totalDuration float64
	botGames     int
	pvpGames     int
}

func NewAnalytics() *Analytics {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "game-events",
		GroupID:  "analytics-group",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &Analytics{
		reader: reader,
	}
}

func (a *Analytics) Start() {
	log.Println("🎮 Analytics Consumer Started")
	log.Println("=============================")
	log.Println("Listening for game events...")
	log.Println("")

	for {
		ctx := context.Background()
		msg, err := a.reader.ReadMessage(ctx)
		if err != nil {
			log.Println("❌ Error reading message:", err)
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Println("❌ Error parsing event:", err)
			continue
		}

		a.processEvent(event)
	}
}

func (a *Analytics) processEvent(event map[string]interface{}) {
	eventType, ok := event["event_type"].(string)
	if !ok {
		return
	}

	timestamp := event["timestamp"]

	switch eventType {
	case "game_start":
		a.gamesStarted++
		gameID := event["game_id"]
		player1 := event["player1"]
		player2 := event["player2"]
		isBot := event["is_bot"]

		if isBot == true {
			a.botGames++
			log.Printf("📊 GAME START (Bot)")
		} else {
			a.pvpGames++
			log.Printf("📊 GAME START (PvP)")
		}
		
		log.Printf("   Game ID: %v", gameID)
		log.Printf("   Players: %v vs %v", player1, player2)
		log.Printf("   Time: %v", timestamp)
		log.Println("")

	case "game_end":
		a.gamesEnded++
		gameID := event["game_id"]
		winner := event["winner"]
		duration, _ := event["duration"].(float64)
		a.totalDuration += duration

		log.Printf("🏆 GAME END")
		log.Printf("   Game ID: %v", gameID)
		log.Printf("   Winner: %v", winner)
		log.Printf("   Duration: %.2f seconds", duration)
		log.Println("")

		// Print statistics every 5 games
		if a.gamesEnded%5 == 0 {
			a.printStats()
		}
	}
}

func (a *Analytics) printStats() {
	log.Println("📈 ===== STATISTICS =====")
	log.Printf("   Total Games Started: %d", a.gamesStarted)
	log.Printf("   Total Games Ended: %d", a.gamesEnded)
	log.Printf("   Bot Games: %d", a.botGames)
	log.Printf("   PvP Games: %d", a.pvpGames)
	if a.gamesEnded > 0 {
		avgDuration := a.totalDuration / float64(a.gamesEnded)
		log.Printf("   Average Game Duration: %.2f seconds", avgDuration)
	}
	log.Println("========================")
	log.Println("")
}

func main() {
	analytics := NewAnalytics()

	// Print stats every 60 seconds
	go func() {
		for {
			time.Sleep(60 * time.Second)
			analytics.printStats()
		}
	}()

	analytics.Start()
}