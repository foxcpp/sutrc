# sutserver
Remote control system for SUT university - task dispatcher daemon.

### Server configuration

**Use corresponding build tag to enable support for your SQL database!**
Build with `postgresql` tag to get PostgreSQL support.
Build with `mysql` tag to get MySQL support.
Support for SQLite3 is included by default, use `nosqlite3` tag to disable it.

Just compile it using regular Go tools and run it as follows:
```
sutserver server 8000 DRIVER=DSN STORAGE
```
DRIVER is driver to use for SQL DB (same as build tag you used to enable it).
DSN is Data Source Name, see underlying driver documentation for exact format you should use:
- PostgreSQL https://godoc.org/github.com/lib/pq
  TLDR: `postgres://user:password@address/dbname`
- MySQL https://github.com/go-sql-driver/mysql
  TLDR: `username:password@protocol(address)/dbname`
- SQLite3 https://github.com/mattn/go-sqlite3
  TLDR `filepath`

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
