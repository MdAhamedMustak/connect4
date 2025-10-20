package main

import (
	"testing"
)

func TestCheckWinner(t *testing.T) {
	gs := &GameServer{
		games: make(map[string]*GameState),
	}

	game := &GameState{
		Board: make([][]Color, ROWS),
	}
	for i := range game.Board {
		game.Board[i] = make([]Color, COLS)
	}

	// Test horizontal win
	game.Board[5][0] = Red
	game.Board[5][1] = Red
	game.Board[5][2] = Red
	game.Board[5][3] = Red

	if !gs.checkWinner(game, 5, 3) {
		t.Error("Failed to detect horizontal win")
	}

	// Test vertical win
	game = &GameState{
		Board: make([][]Color, ROWS),
	}
	for i := range game.Board {
		game.Board[i] = make([]Color, COLS)
	}

	game.Board[2][0] = Yellow
	game.Board[3][0] = Yellow
	game.Board[4][0] = Yellow
	game.Board[5][0] = Yellow

	if !gs.checkWinner(game, 5, 0) {
		t.Error("Failed to detect vertical win")
	}

	// Test diagonal win
	game = &GameState{
		Board: make([][]Color, ROWS),
	}
	for i := range game.Board {
		game.Board[i] = make([]Color, COLS)
	}

	game.Board[2][0] = Red
	game.Board[3][1] = Red
	game.Board[4][2] = Red
	game.Board[5][3] = Red

	if !gs.checkWinner(game, 5, 3) {
		t.Error("Failed to detect diagonal win")
	}
}

func TestBotMove(t *testing.T) {
	gs := &GameServer{
		games: make(map[string]*GameState),
	}

	game := &GameState{
		Board: make([][]Color, ROWS),
	}
	for i := range game.Board {
		game.Board[i] = make([]Color, COLS)
	}

	// Bot should block winning move
	game.Board[5][0] = Red
	game.Board[5][1] = Red
	game.Board[5][2] = Red

	col := gs.getBotMove(game)
	if col != 3 {
		t.Errorf("Bot should block at column 3, got %d", col)
	}

	// Bot should take winning move
	game = &GameState{
		Board: make([][]Color, ROWS),
	}
	for i := range game.Board {
		game.Board[i] = make([]Color, COLS)
	}

	game.Board[5][0] = Yellow
	game.Board[5][1] = Yellow
	game.Board[5][2] = Yellow

	col = gs.getBotMove(game)
	if col != 3 {
		t.Errorf("Bot should win at column 3, got %d", col)
	}
}

func TestIsBoardFull(t *testing.T) {
	gs := &GameServer{}

	game := &GameState{
		Board: make([][]Color, ROWS),
	}
	for i := range game.Board {
		game.Board[i] = make([]Color, COLS)
	}

	if gs.isBoardFull(game) {
		t.Error("Empty board should not be full")
	}

	// Fill board
	for i := 0; i < ROWS; i++ {
		for j := 0; j < COLS; j++ {
			game.Board[i][j] = Red
		}
	}

	if !gs.isBoardFull(game) {
		t.Error("Full board not detected")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if len(id1) != 8 {
		t.Errorf("ID length should be 8, got %d", len(id1))
	}

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}
}