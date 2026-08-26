## How to run

Run the server:
``make run``

Simulate a client:
``make client``

## Prompt history

Here is the list of the important prompts I used:

### Base Golang application

``I am creating a Go server application. I need you to create the basics of it.

That means having a way to run the app and run the tests for it.

I want to both layers of unit and acceptance tests.``

### gRPC implementation

``This application works as a default http server. I need the application to works on gRPC.

That means I want a dedicated folder which describes all the incoming requests. This folder represents a gateway layer.

You need to prepare the incoming implementation of a router which will then route the different requests to different workers. You don't need to implement this router fully for the moment.

I need this system to be easily testable. That means a test client will be generated. The implementation of this client has to be bound to the two test layers.``