## Base application

We are working on a Go game server application.

This application is aimed to manage millions of players playing Tic-Tac-Toe.

This application has to stay basic and is meant to be a demonstration. That means it is important to have a way to run the app and run the tests for it very fast. We want two layers of tests: unit and acceptance tests. They need to be updated every time a featured is added or modified.

## Tic-Tac-Toe game logic

The board is a configurable `width × height` grid with cells indexed in row-major order starting at 0. The default is a 3×3 grid:

```
0 | 1 | 2
---------
3 | 4 | 5
---------
6 | 7 | 8
```

The grid dimensions and winning length are set at game-creation time via `GridConfig` and remain fixed for the lifetime of that game.

### Rules

- `players[0]` (the game creator) always moves first.
- Players alternate turns. A move consists of choosing an **empty** cell and marking it.
- The first player to align `winning_length` consecutive marks in any row, column, or diagonal wins.
- If all cells are filled with no winner, the game ends in a **draw**.
- No move is accepted after the game is over.

### Win detection

After each mark is placed, the server scans outward from that cell in all four axis directions (horizontal, vertical, diagonal-down, diagonal-up), counting consecutive marks of the same player. If the combined count (including the placed cell) reaches `winning_length` in any direction, that player wins. No static winning-lines table is used.

### Move errors

| Error | Cause |
|---|---|
| `ErrGameNotReady` | Fewer than two players have joined |
| `ErrNotYourTurn` | `player_id` does not hold the current turn |
| `ErrCellOccupied` | The chosen cell is already marked |
| `ErrInvalidCell` | Cell index is outside `0 – (width × height − 1)` |
| `ErrGameOver` | The game has already ended (win or draw) |

### Game outcome in `UpdateGameResponse`

After each successful move, `UpdateGameResponse` carries:

- `board` — `width × height` strings, each either `""` (empty) or the `player_id` who marked that cell.
- `winner` — `""` while the game is ongoing, the winning `player_id` on a win, or `"draw"`.

## gRPC architecture

The server runs on gRPC. All incoming requests are described as protobuf RPCs in `proto/voodoo/v1/voodoo.proto`, which acts as the gateway layer. Go code is generated from that file into `gen/voodoo/v1/` using `make generate` (requires `protoc` and the `protoc-gen-go`/`protoc-gen-go-grpc` plugins on PATH). Never edit files under `gen/` by hand.

Incoming requests are dispatched by `internal/router`, which embeds `UnimplementedVoodooServiceServer`. Adding a new RPC to the proto and wiring it in the router is the standard way to extend the application. Workers are separate packages called by the router.

### Testing the gRPC connection

Both test layers spin up a real gRPC server on a random local port and connect to it through the shared test client in `testclient/client.go`. No mocks or HTTP helpers are used.

```bash
# Run all tests (unit + acceptance)
make test

# Unit tests only (router logic, server wiring)
make test/unit

# Acceptance tests only (full server boot + real gRPC call)
make test/acceptance
```

To test manually against a running server, use [grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
# Start the server
make run

# Use the client script in cmd\client\main.go
make client
```

## Workload division layer

High-volume requests are distributed across Workers by a `Sharder` (`internal/sharder`). Each incoming request carries a `gameId`; the Sharder maps it to a specific Worker and keeps that mapping stable for the lifetime of the process.

### Sharder (`internal/sharder`)

The Sharder holds a fixed pool of Workers and a hash index — a `map[string]*Worker` that stores each seen `gameId` as a direct pointer to its assigned Worker in memory, mirroring the hash index pattern from database systems.

On first resolution, `gameId` is hashed with FNV-1a and mapped to a Worker via `hash % numWorkers`. The result is cached in the index so all subsequent calls for the same `gameId` are a single O(1) map lookup under a shared read lock. The pool size is fixed at construction; resizing would require rehashing the entire index.

The Router (`internal/router`) owns a `*Sharder` instance (8 workers by default). When a future RPC carries a `gameId`, the router resolves the target Worker with `r.sharder.Resolve(req.GameId)`.

### Worker (`internal/worker`)

A Worker is the unit of processing for a subset of `gameId`s. It is identified by an integer ID. A shard is a purely abstract concept — it represents the slice of load a Worker is responsible for and has no concrete representation in code. Business logic inside Workers is not yet implemented.

## Game state system

Each Worker owns a `map[string]*GameState` protected by a `sync.RWMutex`. A `GameState` represents one game session and holds:

- `ID` — the game identifier, generated as `"game-{workerID}-{seq}"` where `seq` is a per-worker atomic counter.
- `players [2]string` — the two player IDs. `players[0]` is set at creation; `players[1]` is filled when the second player joins.
- `playerCount int` — current number of players (0–2), guarded by a per-state `sync.Mutex`.
- `currentTurn int` — index (0 or 1) of the player whose turn it is. Starts at 0 (`players[0]`) and flips on every successful `TakeTurn` call.
- `width`, `height`, `winLen int` — grid dimensions and winning length, fixed at creation from `GridConfig`.
- `board []int8` — row-major slice of `width × height` cells: `0` = empty, `1` = `players[0]`, `2` = `players[1]`.
- `ready chan struct{}` — closed when both slots are filled, allowing any goroutine to block on `GameState.WaitReady(ctx)` until the game becomes full.

A game that has one player is in a **waiting** state. A game with two players is **ready**. The maximum capacity is fixed at two; any further join attempt returns `ErrGameFull`.

### Worker methods

| Method | Description |
|---|---|
| `CreateGame(playerID, cfg)` | Allocates a new `GameState` with `playerID` as the first player and the given `GridConfig`, stores it in the map, and returns it. A zero-value `GridConfig` defaults to a 3×3 board with winning length 3. |
| `JoinGame(gameID, playerID)` | Looks up an existing state and calls `AddPlayer`. Returns `ErrGameFull` if the game already has two players, or an error if the game is not found. |
| `UpdateGame(gameID, playerID)` | Calls `GameState.TakeTurn`. Returns `ErrGameNotReady` if only one player has joined, `ErrNotYourTurn` if it is not `playerID`'s turn, or an error if the game is not found. |
| `Get(gameID)` | Returns the `*GameState` for a given ID, or `nil`. |
| `StateCount()` | Returns the number of game states held by this worker. Used by the Sharder for load balancing. |

## Game endpoints

The four game RPCs are defined in `proto/voodoo/v1/voodoo.proto` and dispatched by the Router.

### GridConfig

```
message GridConfig {
  int32 width = 1;          // number of columns
  int32 height = 2;         // number of rows
  int32 winning_length = 3; // consecutive marks needed to win
}
```

Omitting `GridConfig` (or setting all fields to zero) in a request defaults to a 3×3 board with winning length 3.

### CreateGame

```
rpc CreateGame(CreateGameRequest) returns (CreateGameResponse)
  CreateGameRequest  { string player_id, GridConfig grid }
  CreateGameResponse { string game_id  }
```

1. The Router reads `grid` from the request (defaulting to 3×3, win length 3 if absent) and constructs a `worker.GridConfig`.
2. The Router calls `sharder.PickLeastLoaded()` to select the Worker with the fewest active game states.
3. That Worker creates a new `GameState` with `player_id` and the resolved `GridConfig`, then generates a unique `game_id`.
4. The Router calls `sharder.Register(game_id, worker)` to pin this ID to the chosen Worker in the sharder's index, ensuring all future RPCs for this game route to the same Worker.
5. `game_id` is returned immediately. The game is now in the **waiting** state.

### JoinGame

```
rpc JoinGame(JoinGameRequest) returns (JoinGameResponse)
  JoinGameRequest  { string game_id, string player_id }
  JoinGameResponse { string game_id }
```

The Router resolves the Worker via `sharder.Resolve(game_id)` (O(1) index lookup after registration) and calls `worker.JoinGame`. Returns `FailedPrecondition` if the game is already full.

### UpdateGame

```
rpc UpdateGame(UpdateGameRequest) returns (UpdateGameResponse)
  UpdateGameRequest  { string game_id, string player_id }
  UpdateGameResponse { string game_id }
```

The Router resolves the Worker and calls `worker.UpdateGame(game_id, player_id)`, which enforces turn order via `GameState.TakeTurn`. Returns `NotFound` if the game ID is unknown. Returns `FailedPrecondition` if the game has only one player (`ErrGameNotReady`) or if `player_id` does not hold the current turn (`ErrNotYourTurn`). On success the turn advances to the other player.

### ListPendingGames

```
rpc ListPendingGames(ListPendingGamesRequest) returns (ListPendingGamesResponse)
  ListPendingGamesRequest  {}
  ListPendingGamesResponse { repeated string game_ids }
```

Returns the IDs of all games currently in the **waiting** state (one player, no second player yet). The Sharder maintains a `pending` set — a `map[string]struct{}` updated on every `CreateGame` and `JoinGame` — so this call is O(1) for membership and O(n) in the number of pending games for the list itself. `CreateGame` calls `sharder.AddPending`; a successful `JoinGame` calls `sharder.RemovePending`.

### GetPlayerStats

```
rpc GetPlayerStats(GetPlayerStatsRequest) returns (GetPlayerStatsResponse)
  GetPlayerStatsRequest  { string player_id }
  GetPlayerStatsResponse { int64 wins, int64 losses, int64 draws }
```

Returns the lifetime win/loss/draw counts for a player. Counts are maintained by the Router in an `internal/stats.Store` — a thread-safe in-memory map keyed by `player_id`. The Router updates it at the end of every `UpdateGame` call that ends the game: it records a win and a loss for the two players on a decisive result, or a draw for both on a full board with no winner. Players who have never completed a game return all zeros.
