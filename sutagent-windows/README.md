# sutagent-windows 

Just run built binary. It will self-register on server, make sure you have agents self-registration enabled during agent deployment.

### Configuration

`transport.go` contains two hardcoded variables: Code page used for command
output decoding and API base url. Change them!

There are also several hardcoded paths around in code, search by `C:\sutrc`. You may want to
change them.

Latter can be set during build by using `./build.sh` script.

**Note:** baseURL should specify without `api` suffix.
I.e. it should contain `https://localhost/sutrc` and not `https://localhost/sutrc/api`.

### Self-Update

sutagent can update itself on `"selfupdate"` task by downloading latest version from `BASEURL/sutagent.exe`.
