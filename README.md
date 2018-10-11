# sutrc protocol
Lightweight remote control system for SUT university.

### Background

We have a lot of PCs and we need to execute commands on them from Web interface.
Most of existing solution are heavyweight and proprietary (and so they
require us to pay money for usage). That's how sutrc appeared.

### Roles

#### Dispatcher server
_In [sutserver](sutserver) subdirectory._

Just forwards JSON objects between web clients and agents while
providing simple access control.

#### Web interface (client)
_In [sutcp](sutcp) subdirectory._

Written in JS and fully runs inside your browser. This allows us to not
clutter dispatcher server with form logic and not to create additional
backend server just for this.

#### Agent
_Windows implementation is in [sutagent-windows](sutagent-windows) subdirectory._

Responsive for execution of commands and stuff on target machines.
Constantly polls server for "tasks" to do.

### Concepts

#### Tasks concept

Commands and other things we want to do with target machines are
represented by abstract object, called "task".

Each task have a type tag and numeric ID. First is used to determine what to do
with it on agent side. Second is required to coordinate
asynchronous execution (read on). Task can also have any additional
type-dependant information attached to it.

It's not very useful to pass tasks around without knowing what happened
with them. To provide useful feedback agent is required to send
"task result". This is another abstract concept because server doesn't
enforces certain structure on it (however there is some recommendations,
see below). "Task result" is represented by JSON object too.

#### Task lifetime

Entire system consists of three components:
- Dispatcher server (this repo)
- Web interface (client)
- Agent

Typical task flow:
1. Agent connects to server and waits for it to send task objects.
2. Task gets submitted by client and added to queue for particular agent.
   Server will block request until task result is received from agent,
   thus creating illusion of synchronous interaction for client.
3. Server forwards task to client.
4. Agent parses task object and decides what to do.
   Note that it's recommended for agent implementation to execute
   tasks asynchronously and return to listening as fast as possible.
5. When agent is done with task it pushes result to the server.
6. Server forwards result to client who initially submitted task.

#### Authorization

Server provides means to prevent unauthorized access to agents through
it. Clients are required to "log in" and get session token before
doing anything. Agents authorize using simplified flow to make
implementation simpler. They just pass "secret token" with each
request.

#### Communication protocol

HTTP with JSON payloads is used for all I/O.
Both protocols are wide-spread and have implementations almost
everywhere (this is basically why we have client just in browser
instead of client+server).

Task object is required to contain only two fields: `"id"` for **numeric**
Task ID and `"type"` for type (represented as string). All other fields
depend on task type (see end of this file for description of few
"official" task types).

Some HTTP-requests block sender for relatively long time
(aka "longpolling"). These requests will time out and return nothing
(usually empty JSON object without error because time out here is usual
part of operation). Timeout is 26 seconds unless otherwise is said.

Most non-GET requests return empty body (even without JSON object)
unless error is happened.

Generic schema for all errors:
```
{
 "error": true,
 "msg": "human-readable description"
}
```
Non-error responses format varies depending on endpoint.
HTTP status code is **always different from 200** if error is returned.

#### HTTP API reference

Can be found in [HTTP_API.md](HTTP_API.md) file.
