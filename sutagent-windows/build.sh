#!/bin/sh

if [ $# -ne 1 ]; then
    echo "Usage: $0 baseURL"
    exit 1
fi

export GOOS=windows
go build -o sutagent.exe -ldflags "-X main.baseURL=$1 -H windowsgui"
