# sutserver
Remote control system for SUT university - task dispatcher daemon.

### Server configuration

Just compile it using regular Go tools (make sure you have C compiler
because we use SQLite) and run it as follows:
```
sutserver server 8000 database_path
```
`8000` is port to listen on. It's recommended to use reverse proxy
because sutserver lacks TLS support.
Database with all stuff will be stored in spcified file (`database_path`). Note that however
two temporary files will be created during server execution.
`database_path-wal` and `database_path-shm`. Both **should be copied too**
if you are moving database somewhere else. They usually will be deleted
on **normal** server shutdown.

If you are using using HTTPS (what you should do of course) then you need to 
configure your reverse proxy to pass "X-HTTPS-Downstream: 1" and Host headers,
otherwise file downloading from agents may work incorrectly (see
[filedrop](https://github.com/foxcpp/filedrop) documentation for details).

Here is example of correct configuration for nginx with sutserver running at 8000 port:
```
location ~ /sutrc/api {
    proxy_pass http://127.0.0.1:8000;
    proxy_set_header X-HTTPS-Downstream "1";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

### systemd unit

`sutserver@.service` is provided for convenience.
Copy it to `/etc/systemd/systemd` and start like this:
```
systemctl start sutserver@8000
```
Where `8000` is port to listen on. Note that specifing different port will make
it use different database file (`/var/lib/sutserver-PORT/auth.db`).

### Command-line utility how-to

Server binary also acts as a console utility for database maintenance.
