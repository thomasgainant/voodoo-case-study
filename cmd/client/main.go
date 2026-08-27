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

	// Simulate two turns: player-1 moves, then player-2 moves.
	if _, err = client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: gameID, PlayerId: "player-1"}); err != nil {
		log.Fatalf("UpdateGame (player-1 turn) failed: %v", err)
	}
	log.Printf("[player-1] Made a move in game %q", gameID)

	if _, err = client.UpdateGame(ctx, &pb.UpdateGameRequest{GameId: gameID, PlayerId: "player-2"}); err != nil {
		log.Fatalf("UpdateGame (player-2 turn) failed: %v", err)
	}
	log.Printf("[player-2] Made a move in game %q", gameID)
}
