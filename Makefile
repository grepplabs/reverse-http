.DEFAULT_GOAL := help

.PHONY: clean build fmt test

ROOT_DIR      := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

BUILD_FLAGS   ?=
VERSION       = "develop"
BRANCH        = $(shell git rev-parse --abbrev-ref HEAD)
REVISION      = $(shell git describe --tags --always --dirty)
BUILD_DATE    = $(shell date +'%Y.%m.%d-%H:%M:%S')
LDFLAGS       ?= -w -s \
	-X github.com/grepplabs/reverse-http/config.Version=${VERSION} \
	-X github.com/grepplabs/reverse-http/config.Commit=${REVISION} \
	-X github.com/grepplabs/reverse-http/config.Date=${BUILD_DATE}

BINARY        = reverse-http

TEST_AGENT_ID   = 4711
TEST_AUTH       = ha-tls
TEST_STORE_TYPE = none

default: help

.PHONY: help
help:
	@grep -E '^[a-zA-Z%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build executable
	@CGO_ENABLED=0 GO111MODULE=on go build -mod=vendor -o $(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

test: ## Test
	@GO111MODULE=on go test -count=1 -mod=vendor -v ./...

fmt: ## Go format
	go fmt ./...

vet: ## Go vet
	go vet ./...

clean: ## Clean
	@rm -rf $(BINARY)

lint: ## Lint
	@golangci-lint run

.PHONY: deps
deps: ## Get dependencies
	GO111MODULE=on go get ./...

.PHONY: vendor
vendor: ## Go vendor
	GO111MODULE=on go mod vendor

.PHONY: tidy
tidy: ## Go tidy
	GO111MODULE=on go mod tidy

##### Testing

docker-compose.build:
	docker-compose -f $(ROOT_DIR)/docker-compose.${TEST_AUTH}.yml build

docker-compose.up:
	docker-compose -f $(ROOT_DIR)/docker-compose.${TEST_AUTH}.yml up --remove-orphans

docker-compose.down:
	docker-compose -f $(ROOT_DIR)/docker-compose.${TEST_AUTH}.yml down --remove-orphans

docker-compose.run: docker-compose.build docker-compose.up

start-proxy: build
	@export QUIC_GO_LOG_LEVEL_=debug && ${ROOT_DIR}/reverse-http proxy --store.type="${TEST_STORE_TYPE}" --agent-server.listen-address=":4242" \
		--http-proxy.listen-address=":3128" --agent-server.tls.file.key=tests/cfssl/certs/proxy-key.pem --agent-server.tls.file.cert=tests/cfssl/certs/proxy.pem

start-proxy-tls: build
	@export QUIC_GO_LOG_LEVEL_=debug && ${ROOT_DIR}/reverse-http proxy --store.type="${TEST_STORE_TYPE}" --agent-server.listen-address=":4242" \
		--http-proxy.listen-address=":3128" --agent-server.tls.file.key=tests/cfssl/certs/proxy-key.pem --agent-server.tls.file.cert=tests/cfssl/certs/proxy.pem \
		--http-proxy.tls.enable --http-proxy.tls.file.key=tests/cfssl/certs/proxy-key.pem --http-proxy.tls.file.cert=tests/cfssl/certs/proxy.pem

start-proxy2: build
	@${ROOT_DIR}/reverse-http proxy  --store.type="memcached" --agent-server.listen-address=":4243" --http-proxy.listen-address=":3127" \
		--store.http-proxy-address="localhost:3127" --agent-server.tls.file.key=tests/cfssl/certs/proxy-key.pem --agent-server.tls.file.cert=tests/cfssl/certs/proxy.pem

start-agent: build
	@export QUIC_GO_LOG_LEVEL_=debug && ${ROOT_DIR}/reverse-http agent --auth.noauth.agent-id="4711" --agent-client.server-address="localhost:4242" --agent-client.tls.file.root-ca=tests/cfssl/certs/ca.pem

start-agent2: build
	@${ROOT_DIR}/reverse-http agent --auth.noauth.agent-id="4712" --agent-client.server-address="localhost:4243" --agent-client.tls.insecure-skip-verify

start-lb: build
	@${ROOT_DIR}/reverse-http lb --http-proxy.listen-address=":3129" --store.type="${TEST_STORE_TYPE}"

curl-proxy:
	curl -x "http://${TEST_AGENT_ID}:noauth@localhost:3128" https://httpbin.org/ip

curl-proxy-tls:
	curl -x "https://${TEST_AGENT_ID}:noauth@localhost:3128" https://httpbin.org/ip --proxy-cacert tests/cfssl/certs/ca.pem

curl-lb:
	curl -x "http://${TEST_AGENT_ID}:noauth@localhost:3129" https://httpbin.org/ip

jwt-keys: build
	@${ROOT_DIR}/reverse-http auth key private --out=${ROOT_DIR}/tests/jwt/auth-key-private.pem
	@${ROOT_DIR}/reverse-http auth key public --out=${ROOT_DIR}/tests/jwt/auth-key-public.pem --in=${ROOT_DIR}/tests/jwt/auth-key-private.pem
	@${ROOT_DIR}/reverse-http auth jwt token --duration=87600h --agent-id="4711" --role "client" --out ${ROOT_DIR}/tests/jwt/auth-client-jwt-4711.b64 --in=${ROOT_DIR}/tests/jwt/auth-key-private.pem
	@${ROOT_DIR}/reverse-http auth jwt token --duration=87600h --agent-id="4711" --role "agent" --out ${ROOT_DIR}/tests/jwt/auth-agent-jwt-4711.b64 --in=${ROOT_DIR}/tests/jwt/auth-key-private.pem
	@${ROOT_DIR}/reverse-http auth jwt token --duration=87600h --agent-id="4712" --role "client" --out ${ROOT_DIR}/tests/jwt/auth-client-jwt-4712.b64 --in=${ROOT_DIR}/tests/jwt/auth-key-private.pem
	@${ROOT_DIR}/reverse-http auth jwt token --duration=87600h --agent-id="4712" --role "agent" --out ${ROOT_DIR}/tests/jwt/auth-agent-jwt-4712.b64 --in=${ROOT_DIR}/tests/jwt/auth-key-private.pem

start-proxy-jwt: build
	@${ROOT_DIR}/reverse-http proxy --auth.type="jwt" --auth.jwt.public-key=tests/jwt/auth-key-public.pem \
		--agent-server.listen-address=":4242" --http-proxy.listen-address=":3128" --agent-server.tls.file.key=tests/cfssl/certs/proxy-key.pem --agent-server.tls.file.cert=tests/cfssl/certs/proxy.pem

start-agent-jwt: build
	@$(eval JWT_TOKEN=$(shell cat tests/jwt/auth-agent-jwt-${TEST_AGENT_ID}.b64))
	@${ROOT_DIR}/reverse-http agent --auth.type="jwt" --agent-client.server-address="localhost:4242" --agent-client.tls.file.root-ca=tests/cfssl/certs/ca.pem --auth.jwt.token="file:tests/jwt/auth-agent-jwt-${TEST_AGENT_ID}.b64"

curl-proxy-jwt:
	@$(eval JWT_TOKEN=$(shell cat tests/jwt/auth-client-jwt-${TEST_AGENT_ID}.b64))
	curl -x "http://${TEST_AGENT_ID}:${JWT_TOKEN}@localhost:3128" https://httpbin.org/ip
