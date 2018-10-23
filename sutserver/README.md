# sutserver
Remote control system for SUT university - task dispatcher daemon.

### Server configuration

**Use corresponding build tag to enable support for your SQL database!**
Build with `postgresql` tag to get PostgreSQL support.
Build with `mysql` tag to get MySQL support.
Support for SQLite3 is included by default, use `nosqlite3` tag to disable it.

Just compile it using regular Go tools and run it as follows:
```
sutserver server 8000 CONFIGFILE
```

`CONFIGFILE` is path to configuration file.
See [documented example](sutserver.example.yml) in this repo for
info about what should be in it.

If you are using using HTTPS (what you should do of course) then you
need to configure your reverse proxy to pass "X-HTTPS-Downstream: 1" and
`Host` headers, otherwise file downloading from agents may work
incorrectly (see [filedrop](https://github.com/foxcpp/filedrop)
documentation for details).

Here is example of correct configuration for nginx with sutserver
running at 8000 port (HTTPS is used):
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

`sutserver.service` is provided for convenience.

Copy it to `/etc/systemd/systemd` and start like this:
```
systemctl start sutserver
```

It will use config file from `/etc/sutserver.yml`.

### Command-line utility how-to

Server binary also acts as a console utility for database maintenance.
