# <img src="https://github.com/gatewayd-io/gatewayd/blob/main/assets/gatewayd-logo.png" alt="gatewayd logo" style="height: 48px; width: 48px;"/> GatewayD

GatewayD is a cloud-native database gateway and framework for building data-driven applications. It sits in between your database(s) and your database client(s) and proxies all queries to and their responses from the database. While doing so, it supports the following features:

- Manage authentication, authorization and access
- Support multiple database backends (SQL and NoSQL)
- Manage queries, connections and clusters
- Secure and encrypt everything AMAP with different security protocols
- Observe, monitor, analyze and control traffic
- Transform queries, results, schemas and data
- Inject data into queries and results
- Deploy anywhere: on-premise, SaaS, Cloud, CI/CD and Serverless
- Cache queries and their results
- And a bunch of other things

## Architecture

The architecture of the GatewayD consists of a core part and the plugins. The core includes network and connection management components. Any other functionality in the system are served by plugins. Then there are core plugins, always shipped with each binary and community-supported plugins.

The ultimate goal is to be able to replace everything, even the core parts, with plugins. The plugin system uses the [Go plugin system over RPC](https://github.com/hashicorp/go-plugin) by HashiCorp. Thus, the core and the plugins talk over RPC.

The high-level component architecture diagram is depicted below:

![Architecture diagrams](assets/architecture-diagram-v0.0.1.png)

## Run GatewayD for development

You can build and run gatewayd by running the following commands. You must have Go and git installed.

```bash
git clone git@github.com:gatewayd-io/gatewayd.git && cd gatewayd
❯ make run
go mod tidy && go run main.go run
11:21PM DBG Loading plugin: gatewayd-plugin-test
11:21PM DBG Plugin loaded: gatewayd-plugin-test
11:21PM WRN plugin configured with a nil SecureConfig
11:21PM INF configuring client automatic mTLS
11:21PM DBG starting plugin args=["../gatewayd-plugin-test/gatewayd-plugin-test"] path=../gatewayd-plugin-test/gatewayd-plugin-test
11:21PM DBG plugin started path=../gatewayd-plugin-test/gatewayd-plugin-test pid=9937
11:21PM DBG waiting for RPC address path=../gatewayd-plugin-test/gatewayd-plugin-test
11:21PM INF configuring server automatic mTLS timestamp=2022-12-21T23:21:02.345+0100
11:21PM DBG using plugin version=1
11:21PM DBG plugin address address=/tmp/plugin1231501310 network=unix timestamp=2022-12-21T23:21:02.381+0100
11:21PM DBG Registering hooks for plugin: gatewayd-plugin-test
11:21PM DBG Registering hook: onConfigLoaded
11:21PM WRN Unknown hook type: onPluginConfigLoaded
11:21PM DBG Plugin metadata loaded: gatewayd-plugin-test
11:21PM DBG Created a new logger
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM DBG New client created: 127.0.0.1:5432
11:21PM INF There are 10 clients in the pool
11:21PM DBG Resolved address to 0.0.0.0:15432
11:21PM INF GatewayD is listening on 0.0.0.0:15432
11:21PM DBG Current system soft limit: 1048576
11:21PM DBG Current system hard limit: 1048576
11:21PM DBG Soft limit is not set, using the current system soft limit
11:21PM DBG Hard limit is not set, using the current system hard limit
11:21PM INF GatewayD is running with PID 9931
11:21PM DBG GatewayD is booting...
11:21PM DBG GatewayD booted
```

## Run tests

The server will start listening on the default 15432 port and will proxy any clients that connects to its port to a PostgreSQL server running on port 5432. While the server is running, run the following commands to test the proxy feature(s). You must have Docker and `psql` installed.

```bash
# This run a PostgreSQL server as a Docker container
docker run --rm --name postgres-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres
# This will try to connect to the PostgreSQL server via GatewayD
psql postgresql://postgres:postgres@localhost:15432/postgres
```

You can use `psql` as you used to do before, to query or insert data into PostgreSQL. You can remove the Docker container by stopping it:

```bash
docker stop postgres-test
```

<!--
## Support

The support section.

## Contributing

The contributing section.
-->
