# sutagent-windows 

### Installation

`sutagent-windows install AGENTNAME` to install sutagent as a Windows
service.

#### Other subcommands

##### `sutagent-windows remove`
Uninstall service.

##### `sutagent-windows debug`
Launch agent without installation.

##### `sutagent-windows start`
Start installed service.

##### `sutagent-windows stop`
Stop installed service.

### Configuration

`transport.go` contains two hardcoded variables: Code page used for command
output decoding and API base url. Change them!

Latter can be set during build by using `./build.sh` script.
