# SUT Control Panel (sutcp)

Web client for sutrc protocol.

![Login Window Screenshot](_screenshots/login.png)
![Dashboard Screenshot](_screenshots/dashboard.png)

### Installation

Extract repository contents to place somewhere in your's HTTP server root.
Make sure HTTP API of sutserver is reachable at `api/<endpoint>` (relative
to location of WebUI files).

### Agent Grouping

sutcp uses `-`-separated prefix of agent username for grouping. Agents without `-` in name go into
"unknown" group. This allows for much easier browsing when you have a lot of agents.
