package main

import (
	"context"
	"log"
	"os"
	"time"

	pb "voodoo-case-study/gen/voodoo/v1"
	"voodoo-case-study/testclient"
)

func main() {
	addr := "localhost:8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	client, err := testclient.New(addr)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	health, err := client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		log.Fatalf("Health RPC failed: %v", err)
	}
	log.Printf("Health: status=%q", health.Status)

	created, err := client.CreateGame(ctx, &pb.CreateGameRequest{PlayerId: "player-1"})
	if err != nil {
		log.Fatalf("CreateGame RPC failed: %v", err)
	}
	log.Printf("[player-1] Created game %q — waiting for a second player...", created.GameId)

	// Show that the new game appears in the pending list.
	pending, err := client.ListPendingGames(ctx, &pb.ListPendingGamesRequest{})
	if err != nil {
		log.Fatalf("ListPendingGames RPC failed: %v", err)
	}
	log.Printf("Pending games (%d): %v", len(pending.GameIds), pending.GameIds)

	// playerJoined carries the game_id once the second player has joined.
	playerJoined := make(chan string, 1)

	// Simulate player 2 joining from a separate goroutine.
	go func() {
		time.Sleep(500 * time.Millisecond)
		joined, err := client.JoinGame(ctx, &pb.JoinGameRequest{
			GameId:   created.GameId,
			PlayerId: "player-2",
		})
		if err != nil {
			log.Fatalf("[player-2] JoinGame RPC failed: %v", err)
		}
		log.Printf("[player-2] Joined game %q", joined.GameId)
		playerJoined <- joined.GameId
	}()

	// Player 1 blocks here until the join is confirmed.
	gameID := <-playerJoined
	log.Printf("[player-1] Player 2 joined! Game %q is ready.", gameID)

	// Show that the game is no longer in the pending list.
	pending, err = client.ListPendingGames(ctx, &pb.ListPendingGamesRequest{})
	if err != nil {
		log.Fatalf("ListPendingGames RPC failed: %v", err)
	}
	log.Printf("Pending games after join (%d): %v", len(pending.GameIds), pending.GameIds)

	// Simulate a full game: player-1 wins the top row (0,1,2), player-2 takes 3,4.
	type move struct {
		player string
		cell   int32
	}
	moves := []move{
		{"player-1", 0},
		{"player-2", 3},
		{"player-1", 1},
		{"player-2", 4},
		{"player-1", 2},
	}
	for _, m := range moves {
		resp, err := client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: gameID, PlayerId: m.player, Cell: m.cell})
		if err != nil {
			log.Fatalf("UpdateGame (%s, cell %d) failed: %v", m.player, m.cell, err)
		}
		log.Printf("[%s] Marked cell %d — board: %v  winner: %q", m.player, m.cell, resp.Board, resp.Winner)
	}

	for _, playerID := range []string{"player-1", "player-2"} {
		s, err := client.GetPlayerStats(ctx, &pb.GetPlayerStatsRequest{PlayerId: playerID})
		if err != nil {
			log.Fatalf("GetPlayerStats (%s) failed: %v", playerID, err)
		}
		log.Printf("[%s] stats — wins: %d  losses: %d  draws: %d", playerID, s.Wins, s.Losses, s.Draws)
	}

	// --- 4×4 game, win length 4 ---
	log.Printf("\n[4×4 game] player-3 creates a 4×4 game with win length 4")
	created4, err := client.CreateGame(ctx, &pb.CreateGameRequest{
		PlayerId: "player-3",
		Grid:     &pb.GridConfig{Width: 4, Height: 4, WinningLength: 4},
	})
	if err != nil {
		log.Fatalf("[4×4] CreateGame failed: %v", err)
	}
	log.Printf("[player-3] Created 4×4 game %q", created4.GameId)

	joined4, err := client.JoinGame(ctx, &pb.JoinGameRequest{GameId: created4.GameId, PlayerId: "player-4"})
	if err != nil {
		log.Fatalf("[4×4] JoinGame failed: %v", err)
	}
	log.Printf("[player-4] Joined 4×4 game %q", joined4.GameId)

	// player-3 wins top row: cells 0,1,2,3; player-4 takes cells 4,5,6
	type move4 struct {
		player string
		cell   int32
	}
	moves4 := []move4{
		{"player-3", 0}, {"player-4", 4},
		{"player-3", 1}, {"player-4", 5},
		{"player-3", 2}, {"player-4", 6},
		{"player-3", 3},
	}
	for _, m := range moves4 {
		resp, err := client.UpdateGame(ctx, &pb.UpdateGameRequest{
			GameId: created4.GameId, PlayerId: m.player, Cell: m.cell,
		})
		if err != nil {
			log.Fatalf("[4×4] UpdateGame (%s, cell %d) failed: %v", m.player, m.cell, err)
		}
		log.Printf("[%s] Marked cell %d — board size: %d  winner: %q",
			m.player, m.cell, len(resp.Board), resp.Winner)
	}
}
