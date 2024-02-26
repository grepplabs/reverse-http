# reverse-http
[![Build Status](https://github.com/grepplabs/reverse-http/actions/workflows/build.yml/badge.svg)](https://github.com/grepplabs/reverse-http/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Release](https://img.shields.io/github/v/release/grepplabs/reverse-http?sort=semver)](https://github.com/grepplabs/reverse-http/releases)

Reverse HTTP proxy over QUIC protocol ([RFC 9000](https://datatracker.ietf.org/doc/html/rfc9000)).

## Architecture

### Standalone

<p style="text-align: center;"><img src="docs/reverse-http-arch.svg" alt="Architecture"></p>

* Agent connection process
  * An agent initiates a connection to the proxy server utilizing the `QUIC` protocol.
    * The connection between the agent and the proxy is persistent
  * Upon connection, the proxy server performs an agent authentication
  * The proxy keeps track of agents' connections
    * Each agent is uniquely identified by an `agentID`
    * Multiple agents can simultaneously connect to the proxy.
    * Only one connection per `agentID` is allowed.

* Client connection process
  * Clients establish a connection with the HTTP proxy by issuing an `HTTP CONNECT` request. This standard method allows the client to specify the desired destination.
  * During the connection process, the proxy authenticates the connecting client using basic `Proxy-Authorization`, where the `username` is utilized to specify the `agentID` that the client wishes to connect to.
  * Once authenticated, the proxy server locates the corresponding agent's `QUIC` connection that is already being tracked.
  * Proxy opens a new `QUIC` stream to the agent and sends all subsequent data through it
  * The agent proceeds with the `CONNECT` procedure by establishing a new TCP connection to the requested destination.

### HA setup

<p style="text-align: center;"><img src="docs/reverse-http-ha.svg" alt="HA"></p>

* Agent connection process
  * An agent initiates a connection to the UDP load balancer, which in turn establishes a connection with one of the proxy servers
  * Upon establishing a connection, the proxy server records an entry in `memcached` for an agentID along with its own HTTP proxy address.
* Client connection process
  * Clients connect to the TCP load balancer, which then establishes a connection with one of the LB servers.
  * Upon connection, the LB server retrieves the HTTP proxy address and an agentID from Memcached.
    * The LB server then sends an `HTTP CONNECT` request to the proxy.

## Install binary release

1. Get latest tag version
    ```
    TAG=$(curl -Ls -o /dev/null -w %{url_effective} https://github.com/grepplabs/reverse-http/releases/latest | grep -o 'tag/.*' | sed 's/tag\///')
    VERSION=$(echo -n $TAG | grep -o 'v[0-9.]*' | sed 's/v//')
    ```
2. Download the release
   * Linux
     ```
      curl -Ls https://github.com/grepplabs/reverse-http/releases/download/${TAG}/reverse-http_${VERSION}_linux_amd64.tar.gz | tar xz
     ```
   * MacOS Intel
     ```
     curl -Ls https://github.com/grepplabs/reverse-http/releases/download/${TAG}/reverse-http_${VERSION}_darwin_amd64.tar.gz | tar xz
     ```

   * MacOS Apple Silicon
     ```
     curl -Ls https://github.com/grepplabs/reverse-http/releases/download/${TAG}/reverse-http_${VERSION}_darwin_arm64.tar.gz | tar xz
     ```

3. Move the binary to a file location on your system PATH.
    ```
    mv ./reverse-http /usr/local/bin/reverse-http
    ```

## Docker run
  ```
  docker run --rm ghcr.io/grepplabs/reverse-http:latest --help
  ```

## Build
### build binary

    make clean build


## Quick requirements

https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes

```bash
sudo bash -c 'echo net.core.rmem_max=2500000 >> /etc/sysctl.conf'
sudo bash -c 'echo net.core.wmem_max=2500000 >> /etc/sysctl.conf'
sudo sysctl -p
```



## Local test standalone

### no auth

```bash
make start-proxy
make start-agent
curl -x "http://4711:noauth@localhost:3128" https://httpbin.org/ip
```

### jwt auth

```bash
make start-proxy-jwt
make start-agent-jwt
make curl-proxy-jwt
```

## Local test docker-compose

```bash
make TEST_AUTH=noauth docker-compose.run
make TEST_AGENT_ID=4711 curl-proxy
make TEST_AGENT_ID=4712 curl-proxy
```

## Whitelisting patterns

```
localhost
localhost:80
localhost:1000-2000
*.zone
*.zone:80
*.zone:1000-2000
127.0.0.1
127.0.0.1:80
127.0.0.1:1000-2000
10.0.0.1/8
10.0.0.1/8:80
10.0.0.1/8:1000-2000
1000::/16
1000::/16:80
1000::/16:1000-2000
[2001:db8::1]/64
[2001:db8::1]/64:80
[2001:db8::1]/64:1000-2000
2001:db8::1
[2001:db8::1]
[2001:db8::1]:80
[2001:db8::1]:1000-2000
```
