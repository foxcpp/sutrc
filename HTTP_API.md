# HTTP API reference

#### Session management

##### `GET /login?token=PASS`
Initiate session for specified user. Result contains

**Response**
```json
{
 "error": false,
 "token": "..."
}
```

##### `POST /logout`
Pass session token returned by `/login` in `Authorization` header to terminate
session.

#### Admin-level

Pass session token returned by `/login` in `Authorization` header.

#### `GET /agents`

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

#### `PATCH /agents?id=OLDID&newId=NEWID`

Rename change name of agent with name OLDID to NEWID.

#### `POST /tasks?target=AGENTID`
**Longpooling endpoint.**

Enqueue task for agent `AGENTID` and wait for result from agent.
Pass event object in request body.

You can override default result waiting timeout (26 seconds) by passing
different value in query string,
like: `POST /tasks?target=AGENTID&timeout=60` will wait for a
minute instead of 26 seconds.

**Response**
```json
{
    "error": true or false,
    other result object fields (depending on task type)
}
```

Note that `error=true` can be set by agent.

#### Agents self-registration

Agents self-registration mode allows agents to automatically create
accounts for themselves, making mass deployment a lot easier.

##### `POST /agents?token=PASS`

Called by client to create new agent account.
Works only if `GET /agents_selfreg` returns 1.

You don't need to supply `Authorization` header.

You can replace other's agent (or your own if you want to change password)
accounts but can't replace admin accounts using this endpoint, in this
case your attempt will be ignored without error.

##### `POST /agent_selfreg?enabled=1`

Allow previous endpoint to be used. `enabled=0` undoes
effect of previous request with `enabled=1`.

##### `GET /agent_selfreg`

Get current status of agent self-registration.

**Response**
Just digit (not in JSON), 1 for enabled, 0 for disabled.

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

#### `POST /task_result?id=TASK_ID`

Report task execution result back to server.
`TASK_ID` - ID of corresponding task. Result object should be passed in
request body.

Agent should use standard error reporting structure to report errors happened
during task execution:
```
{
    "error": true,
    "msg": "Unknown command: taskkill"
}
```

### Pre-defined task types

#### Shell command execution

**JSON type string**: `"execute_cmd"`.

Agent should execute shell command passed in `"cmd"` field of task JSON object and return
result containing `"status_code"` and `"output"` with process status code (see OS documentation) and
copy of stdout (it's allowed to trim it if it exceeds over 5 KB in size).

**Example:**
Task object:
```
{
    "id": 2343
    "type": "execute_cmd",
    "cmd": "echo hello"
}
```
Task result object:
```
{
    "status_code": 0,
    "output": "hello"
}
```

#### Task list query

**JSON type string:** `"proclist"`

Agent should return list of OS processes running on it's machine as JSON array in `"procs"` field 
of response.

Each entry should have `"id"` as numeric process identifier and `"name"` as a human-friendly process
name (usually program binary name).

**Example:**
Task object:
```
{
    "id": 234,
    "type": "proclist"
}
```

Task result object:
```
{
    "procs": [
        {
            "id": 7,
            "name": "chrome.exe"
        },
        {
            "id": 172,
            "name": "hl2.exe"
        }
    ]
}
```

#### Directory contents query

**JSON type string:** `"dircontents"`

Agent should return contents of filesystem directory specified in `"dir"` field.
`"dir"` is always an absolute path.

**Example**
Task object:
```
{
    "id": 2343,
    "dir": "C:\Windows\system32"
}
```

Task result object:
```
{
    "contents": [
        {
            "name": "explorer.exe",
            "dir": false,
            "fullpath": "C:\Windows\system32\explorer.exe"
        },
        {
            "name": "drivers",
            "dir": true,
            "fullpath": "C:\Windows\system32\drivers"
        },
        ...
    ]
}
```

#### Download file request

**JSON type string:** `"downloadfile"`

Agent should download file from location specified by `"url"` field and save it to path
in `"out"` field. Task result should be empty unless error is happened (in this
case use standard error reporting scheme).

It's recommended for clients to increase default result waiting timeout
to give agent enough time to download file.

**Example:**
Task object:
```
{
    "id": 2344,
    "type": "downloadfile",
    "url": "http://.../sutrc/filedrop/5cb1f372-ced2-11e8-9ce3-b083fe9824ac/hosts"
    "out": "C:\\Windows\\system32\\drivers\\etc\\hosts"
}
```

Task result object:
```
{}
```

#### Upload file request

**JSON type string:** `"uploadfile"`

This is reverse of download file request. Agent should upload file from location specified by
`"path"` and return URL assigned by server in result object (see
[filedrop](github.com/foxcpp/filedrop) server documentation for details).

It's recommended for clients to increase default result waiting timeout
to give agent enough time to upload file.

**Filedrop server limits**
Max link uses: 5
Max store time: 1 hour
Max file size: 32 MiB

**Example**
Task object:
```
{
    "id": 2345,
    "type": "uploadfile",
    "path": "C:\\Windows\\system32\\drivers\\etc\\hosts"
}
```

Task result object:
```
{
    "url": "http://.../sutrc/filedrop/5cb1f372-ced2-11e8-9ce3-b083fe9824ac/hosts"
}
```