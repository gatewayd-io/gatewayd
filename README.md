# GatewayD

GatewayD is a cloud-native database gateway. It sits in between your database(s) and your database client(s) and proxies all queries to and their responses from the database. While doing so, it supports the following features:

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

<!--
## Support

The support section.

## Contributing

The contributing section.
-->
