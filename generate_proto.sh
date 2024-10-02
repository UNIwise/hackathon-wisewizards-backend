#!/bin/bash

docker run --rm --user $(id -u):$(id -g) -v $(pwd):$(pwd) -w $(pwd) rvolosatovs/protoc --go_out=. --go-grpc_out=. -I. ./pkg/proto/**/*.proto