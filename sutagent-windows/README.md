# sutagent-windows 

### Installation

`sutagent-windows install` to install sutagent as a Windows service.

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

**Note:** baseURL should specify without `api` suffix.
I.e. it should contain `https://localhost/sutrc` and not `https://localhost/sutrc/api`.


### Self-Update

Compiled updater binary should be located at `BASEURL/sutupdate.exe` and latest version of
agent's binary should be located at `BASEURL/sutagent.exe`.
