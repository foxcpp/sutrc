# SUT Control Panel (sutcp)

Web client for sutrc protocol.

![Login Window Screenshot](_screenshots/login.png)
![Dashboard Screenshot](_screenshots/dashboard.png)

### Installation

HTML/JS files in this repository have placeholders instead of strings. 
You should run Python stript `langsubst.py` to convert them to usable files.
Do it like this:
```
./langsubst.py en.yml
```
This is how we implement static translation support.

This script will create `out` directory, copy it's contents to place 
somewhere where your HTTP server can reach it. 
Make sure sutserver's HTTP API is reachable at `api/` (relative to location of
sutcp files). I.e. if you put sutcp at `http://server/sutcp/dashboard.html`
then API should be reachable at `http://server/sutcp/api`.

### Agent Grouping

sutcp uses `-`-separated prefix of agent username for grouping. Agents without `-` in name go into
"unknown" group. This allows for much easier browsing when you have a lot of agents.
