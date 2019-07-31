#!/bin/bash

set -e -x -u

go fmt ./cmd/...

go build -o controller ./cmd/controller/...
ls -la ./controller

ytt -f config-certmanager/config.yml --file-mark config.yml:type=yaml-plain -f config-certmanager/patches.yml >/dev/null

ytt -f config/ >/dev/null

echo SUCCESS
