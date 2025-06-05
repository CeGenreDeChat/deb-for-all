# Makefile for deb-for-all project

.PHONY: all build clean test

all: build

build:
	go build -o bin/deb-for-all ./cmd/deb-for-all

clean:
	go clean
	rm -rf bin/*

test:
	go test ./... -v

run: build
	./bin/deb-for-all

install:
	go install ./cmd/deb-for-all