#!/bin/bash

# Syntax: ./build.sh hashcalcmt 0.0.1xg-20250126

APP_REALM="criticalsys.net"
APP_NAME="$1"
APP_VERSION="$2"
APP_MODULE="${APP_REALM}/${APP_NAME}"

mkdir -p bin
if [ ! -s go.mod ];then
  echo "Initialize Go Module ${APP_MODULE}"
  go mod init ${APP_MODULE}
fi
echo "Generate dependency list"
go mod tidy
echo "Update all modules to latest release"
go get -u ./...
echo "Build Module ${APP_MODULE} version ${APP_VERSION}"
go build -v -ldflags "-s -w -X main.Version=${APP_VERSION}" -o ./bin/

chmod 644 *.go go*
chmod 755 bin/*

if [ "$3" == "--publish" ];then
  tar Jcvf ../${APP_NAME}-${APP_VERSION%xg*}-amd64.tar.xz *
fi
