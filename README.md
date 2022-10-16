# GatewayD

GatewayD is a cloud-native database gateway and framework. It acts like an API gateway, but instead sits between the applications or microservices, a.k.a. database clients, and proxies connections to the database(s). The database clients are drivers, OXMs, libraries, migration runners, CLIs, UIs, and whatever tries to connect to the database via its wire protocol. The core proxies connections between the client(s) and the database(s), while providing connection pooling, bouncing, and a few other network-level and socket-level features. It also provides a plugin framework, so that the rest of the functionality are implemented on top as plugins. So, whatever happens inside the core needs to be accessible by plugins. Query processing, AAA, security, traffic control, o11y, transformations, logging, and other functionalities are implemented with plugins. There will be core and community plugins. There will be a package manager to manage and update all the plugins. These are the features:

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

The ultimate goal is to be able to replace everything, even the core parts, with plugins. The plugin system uses the [Go plugin system over RPC](https://github.com/hashicorp/go-plugin) by HashiCorp. Thus, the core and the plugins talk over RPC and they can also be run over the network using gRPC or netRPC.

The high-level component architecture diagram is depicted below:

![Architecture diagrams](assets/Architecture%20Diagram%20v0.0.1.svg)

## Build and Run

You can build and run gatewayd by running the following commands. You must have Go and git installed.

```bash
git clone git@github.com:gatewayd-io/gatewayd.git && cd gatewayd
go mod tidy && go build
./gatewayd
Listening on host: ::, port: 15432
```

The server will start listening on the default 15432 port and will proxy any clients that connects to its port to a PostgreSQL server running on port 5432. While the server is running, run the following commands to test the proxy feature(s). You must have Docker and `psql` installed.

```bash
# This run a PostgreSQL server as a Docker container
docker run --rm --name postgres-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres
# This will try to connect to the PostgreSQL server via gatewayd
psql -U postgres -p 15432 -h localhost
```

After entering the password, `postgres`, you can see the messages being passed between the PostgreSQL and the `psql` client in the terminal the gatewayd is running in. The `psql` command will work normally.

You can remove the Docker container by stopping it:

```bash
docker stop postgres-test
```

<!--
## Support

The support section.

## Contributing

The contributing section.
-->
