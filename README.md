# dutserver

...

### Concepts

Server is responsive for events logging and task dispatching, where
event is notification about something happened on agent's side and
task is something agent should do.

Both events and tasks are described by arbitrary JSON objects. Server
ignores contents and just forwards them (for now). This allows to
define new types of these structures without changing server code at
all.

#### Accounts

Agent must be registered on server using command-line utility (see
below) before it will be able to interact with it. Account information
is just name and password pair. Name is used to refer to agent in API
calls and must be unique.

In order to send tasks and view logged events you need a "admin" account.
It is created in way similar to agent's account, just change "agent" to
"admin" in command.

### Command-line utility

```
dutserver server 8000 test.db
```
Start API server on port `8000` using DB in file `test.db`.

```
dutserver addaccount test.db agent1 agent
```
Create agent account `agent1` in DB `test.db`.

```
dutserver remove test.db agent1
```
Remove account `agent1` from DB stored in file `test.db`.


### HTTP API reference

Parameters are passed in URL query. Server responds using JSON in body.

Generic schema for API errors:
```
{
 "error": true,
 "msg": "human-readable description"
}
```
Non-error responses format varies depending on endpoint but always contains
`"error": false`.

#### Session management

##### `POST /login?user=NAME&pass=PASS`
Initiate session for specified user.

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

Returns known agent names.

**Response**
```json
{
 "error": false,
 "agents": {
  "agent1",
  "agent2",
  ...
 }
}
```

##### `GET /events?agent=AGENTID`

Get events from agent `AGENTID`.

**Response**
```json
{
 "error": false,
 "events": [
   ...
 ],
 "max_size": 512
}
```

#### `POST /tasks?target=AGENTID`

**Longpooling endpoint. Timeout is 26 seconds.**

Enqueue task for agent `AGENTID` and wait for result from agent.
Pass event object in request body.

Pass `noWait=1` to disable longpooling and just return instantly.

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

**Response**
Empty, unless error is happened.

##### `POST /agent_selfreg?enabled=1`

Allow previous endpoint to be used. `enabled=0` undoes
effect of previous request with `enabled=1`.

**Response**
Empty, unless error is happened.

##### `GET /agent_selfreg`

Get current status of agent self-registration.

**Response**
Digit, 1 for enabled, 0 for disabled.

#### Agent-level

Agents don't require session to operate and instead just pass
user:pass pair in `Authorization` header.

#### `GET /tasks`

**Longpooling endpoint. Timeout is 26 seconds.**

Return tasks for agent, if any.

**Response**
```
{
 "events": {
  "EVENT_ID": {
   ...
  }
 }
}
```

#### `POST /events`

Notify server about event.
Pass event object in request body.

**Response**
```
{
 "error": false,
 "msg": "Event logged"
}
```

#### `POST /tasks_result?id=TASK_ID`

Report task execution result back to server.
Pass result JSON object in request body.

**Response**
Empty, unless error is happened.