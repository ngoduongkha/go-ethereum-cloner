GIT_COMMIT:=$(shell git rev-list -1 HEAD)
LDFLAGS:=-X main.GitCommit=${GIT_COMMIT}

install:
	go install -ldflags "$(LDFLAGS)" ./...

image:
	docker build -t geth:latest .

local:
	docker compose build
	docker compose run geth-local
