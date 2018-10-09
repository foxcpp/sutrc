# Windows implementation of dutserver agent

### Installation

`dutcontrol-windows install AGENTNAME` to install dutcontrol as a Windows
service.

#### Other subcommands

##### `dutcontrol-windows remove`
Uninstall service.

##### `dutcontrol-windows debug`
Launch agent without installation.

##### `dutcontrol-windows start`
Start installed service.

##### `dutcontrol-windows stop`
Stop installed service.

### Configuration

`transport.go` contains two hardcoded variables: Code page used for command
output decoding and API base url. Change them!
