# sutagent-windows 

### Installation

`sutagent-windows install` to create all necessary files.

Then just run `sutagent-windows` how you like to start agent.

### Configuration

`transport.go` contains two hardcoded variables: Code page used for command
output decoding and API base url. Change them!

Latter can be set during build by using `./build.sh` script.

**Note:** baseURL should specify without `api` suffix.
I.e. it should contain `https://localhost/sutrc` and not `https://localhost/sutrc/api`.

### Self-Update

sutagent can update itself on `"selfupdate"` task by downloading latest version from `BASEURL/sutagent.exe`.
