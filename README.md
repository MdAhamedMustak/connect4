# 🎯 Coonect 4 - Real-time Multiplayer Game

A production-ready, real-time Connect Four game with competitive AI, WebSocket support, PostgreSQL persistence, and Kafka analytics.


## 🌟 Features

- ✅ **Real-time Multiplayer**: WebSocket-based 1v1 gameplay
- 🤖 **Competitive Bot**: Strategic AI opponent with blocking and winning moves
- 🔄 **Auto-matching**: 10-second matchmaking with bot fallback
- 🔌 **Reconnection Support**: 30-second grace period to rejoin games
- 📊 **Live Leaderboard**: Persistent player rankings
- 📈 **Kafka Analytics**: Event-driven game metrics pipeline
- 🎨 **Modern UI**: React with Tailwind CSS
- 🗄️ **PostgreSQL**: Persistent game history and statistics
- 🐳 **Docker Support**: Easy deployment with Docker Compose

## 🚀Github Link - https://github.com/MdAhamedMustak/connect4

## 📋 Prerequisites

- **Go**: 1.21 or higher
- **Node.js**: 16+ and npm
- **PostgreSQL**: 13+ (optional, game works without it)
- **Apache Kafka**: 3.0+ (optional, for analytics)
- **Docker** (optional, for easy deployment)

## 🏗 Architecture

```
┌─────────────┐      WebSocket      ┌─────────────┐
│   Frontend  │ ←─────────────────→ │   Backend   │
│   (React)   │                     │   (GoLang)  │
└─────────────┘                     └──────┬──────┘
                                           │
                     ┌─────────────────────┼────────────────┐
                     │                     │                │
                     ▼                     ▼                ▼
              ┌──────────┐         ┌──────────┐    ┌──────────┐
              │ Postgres │         │  Kafka   │    │Analytics │
              │   (DB)   │         │ (Events) │    │ Consumer │
              └──────────┘         └──────────┘    └──────────┘
```

## 🚀 Quick Start

### Option 1: Docker Setup (Recommended)

```bash
# Clone the repository
git clone <your-repo-url>
cd 4-in-a-row

# Start all services with Docker Compose
docker-compose up -d

# Frontend will be at: http://localhost:3000
# Backend API at: http://localhost:8080
```

### Option 2: Manual Setup

#### 1. Setup PostgreSQL

```bash
# Create databases
createdb connect4
createdb connect4_analytics

# Or using psql
psql -U postgres
CREATE DATABASE connect4;
CREATE DATABASE connect4_analytics;
```

#### 2. Setup Kafka (Optional)

```bash
# Using Docker
docker run -d --name zookeeper -p 2181:2181 zookeeper:3.7
docker run -d --name kafka -p 9092:9092 \
  -e KAFKA_ZOOKEEPER_CONNECT=localhost:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  confluentinc/cp-kafka:latest

# Create topic
kafka-topics --create --topic game-events \
  --bootstrap-server localhost:9092 \
  --partitions 3 --replication-factor 1
```

#### 3. Backend Setup

```bash
cd backend

# Install dependencies
go mod init connect4
go get github.com/gorilla/websocket
go get github.com/lib/pq
go get github.com/segmentio/kafka-go

# Update database connection in main.go if needed
# Default: host=localhost port=5432 user=postgres password=postgres

# Run the server
go run main.go

# Server starts on port 8080
```

#### 4. Analytics Consumer Setup

```bash
cd analytics

# Install dependencies
go mod init analytics
go get github.com/segmentio/kafka-go
go get github.com/lib/pq

# Run the analytics consumer
go run main.go
```

#### 5. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Create .env file
echo "REACT_APP_WS_URL=ws://localhost:8080/ws" > .env
echo "REACT_APP_API_URL=http://localhost:8080" >> .env

# Start development server
npm start

# Frontend runs on http://localhost:3000
```

## 📁 Project Structure

```
4-in-a-row/
├── backend/
│   ├── main.go                 # Main game server
│   ├── go.mod
│   └── go.sum
├── analytics/
│   ├── main.go                 # Kafka analytics consumer
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── App.js             # Main React component
│   │   └── index.js
│   ├── package.json
│   └── public/
├── docker-compose.yml          # Docker setup
├── Dockerfile.backend
├── Dockerfile.analytics
├── Dockerfile.frontend
└── README.md
```

## 🎮 How to Play

1. **Enter Username**: Type your username and click "Join Game"
2. **Wait for Match**: System searches for an opponent (10 seconds max)
3. **Play**: Click columns to drop your disc
4. **Win**: Connect 4 discs horizontally, vertically, or diagonally
5. **Reconnect**: If disconnected, rejoin within 30 seconds using same username

## 🤖 Bot Strategy

The competitive bot uses the following decision hierarchy:

1. **Win immediately** if possible
2. **Block opponent's winning move**
3. **Prioritize center column** (strategic advantage)
4. **Choose adjacent columns** (2, 4, 1, 5, 0, 6 priority)

The bot analyzes the board state and makes strategic decisions, not random moves.

## 📊 API Endpoints

### WebSocket
```
ws://localhost:8080/ws
```

**Messages:**
- `join`: Connect and enter matchmaking
- `move`: Make a move (column 0-6)

### REST API
```
GET /leaderboard - Fetch top players
```

## 📈 Analytics Events

Kafka events published:

### Game Start Event
```json
{
  "event_type": "game_start",
  "game_id": "abc123",
  "player1": "alice",
  "player2": "bob",
  "is_bot": false,
  "timestamp": "2025-10-18T10:30:00Z"
}
```

### Game End Event
```json
{
  "event_type": "game_end",
  "game_id": "abc123",
  "winner": "red",
  "duration": 45.2,
  "is_bot": false,
  "timestamp": "2025-10-18T10:31:00Z"
}
```

## 🔧 Configuration

### Backend (main.go)
```go
// Database connection
connStr := "host=localhost port=5432 user=postgres password=postgres dbname=connect4 sslmode=disable"

// Kafka connection
Addr: kafka.TCP("localhost:9092")
Topic: "game-events"
```

### Frontend (.env)
```env
REACT_APP_WS_URL=ws://localhost:8080/ws
REACT_APP_API_URL=http://localhost:8080
```

## 🐳 Docker Compose

Full stack deployment:

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

  kafka:
    image: confluentinc/cp-kafka:latest
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092

  backend:
    build: ./backend
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - kafka

  analytics:
    build: ./analytics
    depends_on:
      - postgres
      - kafka

  frontend:
    build: ./frontend
    ports:
      - "3000:80"
    depends_on:
      - backend
```

## 📊 Database Schema

### Main Database (connect4)
```sql
CREATE TABLE games (
    id VARCHAR(50) PRIMARY KEY,
    player1 VARCHAR(100) NOT NULL,
    player2 VARCHAR(100) NOT NULL,
    winner VARCHAR(100),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    is_bot BOOLEAN DEFAULT FALSE
);
```

### Analytics Database (connect4_analytics)
```sql
CREATE TABLE game_events (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(50),
    event_data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE game_metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(100),
    metric_value DECIMAL(10, 2),
    recorded_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE player_metrics (
    username VARCHAR(100) PRIMARY KEY,
    games_won INT DEFAULT 0,
    last_win TIMESTAMP
);

CREATE TABLE hourly_games (
    hour TIMESTAMP PRIMARY KEY,
    game_count INT DEFAULT 0
);
```

## 🧪 Testing

### Test WebSocket Connection
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onopen = () => {
  ws.send(JSON.stringify({ type: 'join', username: 'test_user' }));
};
```

### Test API
```bash
curl http://localhost:8080/leaderboard
```

## 🚀 Deployment

### Backend Deployment (Heroku/Railway)
```bash
# Create Procfile
echo "web: ./main" > Procfile

# Deploy
git push heroku main
```

### Frontend Deployment (Vercel/Netlify)
```bash
# Build
npm run build

# Deploy
vercel deploy
```

## 🔒 Security Considerations

- ✅ CORS configured for production domains
- ✅ Input validation on all moves
- ✅ Rate limiting on WebSocket connections
- ✅ SQL injection prevention with parameterized queries
- ⚠️ Add authentication for production
- ⚠️ Use environment variables for secrets

## 🐛 Troubleshooting

### WebSocket Connection Failed
- Check if backend is running on port 8080
- Verify firewall settings
- Check browser console for CORS errors

### Database Connection Error
- Verify PostgreSQL is running: `pg_isready`
- Check connection string credentials
- Ensure databases exist

### Kafka Not Working
- Check if Kafka is running: `docker ps`
- Verify topic exists: `kafka-topics --list`
- Game will work without Kafka (analytics disabled)

### Bot Not Responding
- Check backend logs for errors
- Verify game state is updating
- Bot has 500ms delay (intentional)

## 📝 Future Enhancements

- [ ] User authentication and profiles
- [ ] Game rooms and private matches
- [ ] Elo rating system
- [ ] Game replay feature
- [ ] Chat functionality
- [ ] Tournament mode
- [ ] Mobile app (React Native)
- [ ] Multiple difficulty bot levels

## 👨‍💻 Development

```bash
# Run tests
go test ./...

# Format code
go fmt ./...

# Lint
golangci-lint run
```

## 📞 Support

For issues and questions:
- Create GitHub issue
- Check existing documentation
- Review troubleshooting section

---

Built with ❤️ using Go, React, PostgreSQL, and Kafka
