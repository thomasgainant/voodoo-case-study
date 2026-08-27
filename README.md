## How to run

Run the server:
``make run``

Simulate a client:
``make client``

## Important prompts history

Here is the list of the important prompts I used to accelerate the development:

### Base Golang application

``I am creating a Go server application. I need you to create the basics of it.

That means having a way to run the app and run the tests for it.

I want to both layers of unit and acceptance tests.``

### gRPC implementation

``This application works as a default http server. I need the application to works on gRPC.

That means I want a dedicated folder which describes all the incoming requests. This folder represents a gateway layer.

You need to prepare the incoming implementation of a router which will then route the different requests to different workers. You don't need to implement this router fully for the moment.

I need this system to be easily testable. That means a test client will be generated. The implementation of this client has to be bound to the two test layers.``

### Work load division

``I created a Golang web application using gRPC.

This application is to be designed for high volumes of requests incoming from the Router. I need to implement a structure called Sharder, which lead the different requests to a set of different Workers in charge of a shard, representing a division of the whole data to be processed.

The incoming gRPC requests will all include a unique id, called "gameId" which is used to identify to which shard, meaning in which Worker the incoming request has to be processed.

A shard will be contained inside a Worker, which will then get the incoming data and process according to a business logic which is not yet to be implemented. A shard can correspond to multiple gameIds.

This implies that the Sharder will have a list of instantiated and active Workers.

This also implies that the Sharder will have a table of gameId to which each one will correspond to a pointer to a specific Worker.

This table has to be implemented in the style of Hash Indexes in database systems, with the difference that the indexes won't point to a specific position offset in a file, but pointing directly to the Worker in memory.``

### Game state implementation

``I want to update the Worker structure to describe it as having a collection of game states to create or update as the requests are coming.

Each request can have a gameId which is used to identify which game state to modify.

If the request do not have a gameId, that means we have to create a new game state in memory``

``I want now to update the gRPC API to reflect these two possibilities: create a new game state and modify the game state.

It is important that the use of those two endpoints goes through the router, through the sharder and then up to the dedicated worker.

That means that when using the creation of a game state, no gameId will be involved and a Worker in the pool will have to be picked, according to the lowest work load, aka the smallest game state list.

What the modification of a game state is doing is not yet to be fully implemented, as it will follow specific game rules.

It is important to update the unit tests for this.

It also important to update the test clients in cmd\client\main.go and in testclient\client.go``

``I want to add  a new endpoint which is JoinGame.

That means that, every gameState will have two players id. The JoinGame endpoint will allow a player to join a game, represented by a game state, in which there is only one player id saved.

Once a game has two playerIds in its game state, it is impossible to join it. This needs to be tested in unit tests.

This also means that the CreateGame endpoint implies a waiting time for the player who created the game in which he will wait until someone joins his game through the JoinGame endpoint.``

### Pending games list

``I need now to update the endpoints of the application so a client can search for all pending games. Then the client will be able to join a pending game.

I want the list of pending games to be a list within the Sharder so we can have a O(1) access to the pending games. This list has to be updated once two players have joined a game and that game is no longer pending.

This needs to be tested, in the test environments. but also using cmd\client\main.go``

### Turn system

``The UpdateGame calls should follow a turn based system: one player has a turn, makes a move, and then it is the other player's turn.

If it is not the player's turn and he still tries to make a move, he will receive an error.

This needs to be tested in unit tests as well.``

### Basic Tic-Tac-Toe game logic

``I want the application to describe the rules of Tic-Tac-Toe and especially the case in which players win or lose.

That means the UpdateGame gRPC method will be used for making moves. 

Moves mean a player, whose turn is in, chooses an empty cell inside a Tic-Tac-Toe grid that he will mark as his.

Once a line of marked of cells are aligned, horizontally, vertically or in diagonal, the player wins the game.

Making a move will communicate to both players the new game state.

The basic rules of the game have to be tested in unit tests.``