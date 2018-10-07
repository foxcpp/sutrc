# dutserver
Lightweight remote control system - task dispatcher daemon.

### Background

We have a lot of PCs and we need to execute commands on them from Web interface.
Most of existing solution are heavyweight and proprietary (and so they
require us to pay money for usage). That's how dutserver appeared.

### Roles

#### Dispatcher server

Just forwards JSON objects between web clients and agents while
providing simple access control.

#### Web interface (client)

Written in JS and fully runs inside your browser. This allows us to not
clutter dispatcher server with form logic and not to create additional
backend server just for this.

#### Agent

Responsive for execution of commands and stuff on target machines.
Constantly polls server for "tasks" to do.

### Concepts

#### Tasks concept

Commands and other things we want to do with target machines are
represented by abstract object, called "task".

Each task have a type tag and ID. First is used to determine what to do
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
implementation simpler. They just pass name:secret pair with each
request.

Both clients and agents use the same user:pass pair scheme. Agent's
"username" is used to refer to it in client interface and everywhere
in code, while password (aka agent secret) makes it
impossible for intruder to impersonate agent. Client credentials are
just used to track "who did what" and to prevent random people touching
agents.

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

### HTTP API reference.

#### Session management

##### `GET /login?user=NAME&pass=PASS`
Initiate session for specified user. Result contains

**Response**
```json
{
 "error": false,
 "token": "..."
}
```


##### `POST /logout`
Pass token returned by `/login` in `Authorization` header to terminate
session.

#### Admin-level

Pass session token returned by `/login` in `Authorization` header.

##### `GET /agents`

Returns known agent lists.

Agent is considered online if it currently ready to accept
tasks (listening for them now).

**Response**
```json
{
 "error": false,
 "agents": {
  "agent1",
  "agent2",
  ...
 },
 "online": {
  "agent1": true,
  "agent2": false,
 }
}
```

#### `POST /tasks?target=AGENTID`
**Longpooling endpoint.**

Enqueue task for agent `AGENTID` and wait for result from agent.
Pass event object in request body.

**Response**
```json
{
 "error": false,
 "result": anything
}
```

#### Agents self-registration

Agents self-registration mode allows agents to automatically create
accounts for themselves, making mass deployment a lot easier.

##### `POST /agents?user=NAME&pass=PASS`

Called by client to create new agent account.
Works only if `GET /agents_selfreg` returns 1.

##### `POST /agent_selfreg?enabled=1`

Allow previous endpoint to be used. `enabled=0` undoes
effect of previous request with `enabled=1`.

##### `GET /agent_selfreg`

Get current status of agent self-registration.

**Response**
Just digit, 1 for enabled, 0 for disabled.

#### Agent-level

Agents don't require session to operate and instead just pass
user:pass pair in `Authorization` header.

#### `GET /tasks`
**Longpooling endpoint.**

Return pending tasks for agent, if any.

**Response**
Contains task object that should be "executed" or just empty JSON if
request timed out (agent should just retry in this case).
```
{
 "type": TASK_TYPE,
 "id": TASK_ID,
 ...
}
```

#### `POST /tasks_result?id=TASK_ID`

Report task execution result back to server.
`TASK_ID` - ID of corresponding task. Result object should be passed in
request body.

### Pre-defined task types

### Server configuration

Just compile it using regular Go tools (make sure you have C compiler
because we use SQLite) and run it as follows:
```
dutserver server 8000 database_path
```
`8000` is port to listen on. It's recommended to use reverse proxy
because dutserver lacks TLS support.
Database with all stuff will be stored in spcified file (`database_path`). Note that however
two temporary files will be created during server execution.
`database_path-wal` and `database_path-shm`. Both **should be copied too**
if you are moving database somewhere else. They usually will be deleted
on **normal** server shutdown.

### Command-line utility how-to

Server binary also acts as a console utility for database maintenance.

```
dutserver addaccount test.db agent1 agent
```
Create agent account `agent1` in DB `test.db`. Password will be read
from stdin.

```
dutserver remove test.db agent1
```
Remove account `agent1` from DB stored in file `test.db`.