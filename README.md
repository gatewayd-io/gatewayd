<p align="center">
    <a href="https://gatewayd.io/">
        <picture>
            <source media="(prefers-color-scheme: dark)" srcset="https://github.com/gatewayd-io/gatewayd/blob/main/assets/gatewayd-logotype-dark.png">
            <img alt="GatewayD logotype" src="https://github.com/gatewayd-io/gatewayd/blob/main/assets/gatewayd-logotype-light.png">
        </picture>
    </a>
    <h3 align="center">Like API gateways, for databases</h3>
    <p align="center">Cloud-native database gateway and framework for building data-driven applications.</p>
</p>

<p align="center">
    <a href="https://github.com/gatewayd-io/gatewayd/releases"><img src="https://img.shields.io/github/v/release/gatewayd-io/gatewayd" alt="Downloads" /></a>
    <a href="https://github.com/gatewayd-io/gatewayd/actions/workflows/release.yaml"><img src="https://img.shields.io/github/actions/workflow/status/gatewayd-io/gatewayd/release.yaml" alt="Release Worflow Status" /></a>
    <a href="https://goreportcard.com/report/github.com/gatewayd-io/gatewayd"><img src="https://goreportcard.com/badge/github.com/gatewayd-io/gatewayd" alt="Go Report Card" /></a>
    <a href="https://docs.gatewayd.io/"><img src="https://img.shields.io/badge/read-docs-brightgreen" alt="Documentation"></a>
    <a href="https://pkg.go.dev/github.com/gatewayd-io/gatewayd"><img src="https://pkg.go.dev/badge/github.com/gatewayd-io/gatewayd.svg" alt="Go Reference"></a>
    <a href="https://coveralls.io/github/gatewayd-io/gatewayd?branch=main"><img src="https://coveralls.io/repos/github/gatewayd-io/gatewayd/badge.svg?branch=main" alt="Coverage Status" /></a>
    <a href="https://awesome-go.com/database-tools/"><img src="https://awesome.re/badge.svg" alt="Mentioned in Awesome Go" /></a>
</p>

**GatewayD** is a free and open-source cloud-native database gateway and framework for building data-driven applications. It is a middleware that sits between your database servers and clients and proxies all their communication. It is like API gateways in system design, but instead is used for proxying database traffic, rather than HTTP traffic.

GatewayD is an [L4](https://en.wikipedia.org/wiki/Transport_layer) proxy for SQL, and eventually NoSQL, databases and clients. The core is database-protocol-agnotic, and the plugins encode, decode and add value to the database traffic flow, hence it can technically support all databases. You can either write your own plugins or make use of our built-in, community and enterprise plugins.

Using GatewayD, you can see through the queries and the data passing between your database server and clients, and take action. For example, you can cache the result of SQL SELECT queries or detect and prevent SQL injection attacks.

GatewayD is developed by [GatewayD Labs](https://gatewayd.io/) and the community.

## Features

- **Cloud-native**:

    Built with cloud-native principles in mind: containerized, stateless, <!--, scalable,--> observable and extensible, while being secure and reliable.

- **Connection pooling and proxying**:

    Pools connections to database servers and clients and proxies them together.

- **Database-agnostic**:

    GatewayD proxies connections, while plugins enable database support.

- **Plugin-based & extensible**:

    Plugins extend functionality. You can write your own plugins or use our built-in, community and enterprise plugins.

- **Multi-tenancy**:

    Manage multiple databases and clients within a single GatewayD instance.

- **Full observability**:

    Integrated logging, metrics, and tracing for comprehensive monitoring and observability.

## Plugins & SDK

The plugins are the lifeblood of GatewayD. They are loaded on startup to add tons of functionality, for example:

- **Query parsing and processing**
- **Authentication, authorization and access management**
- **Caching**
- **Injection detection and prevention**
- **Firewalling**
- **Query, schema and data management and transformation**
- **Change data capture**
- **Distributed query processing**
- Many other possibilities

The plugins are *usually* written in Go and are compiled into stand-alone executables. They are loaded on startup and are ready to be used by the users. In turn, the developers can write their own plugins and use them in their applications, which is made possible by using the [SDK](https://github.com/gatewayd-io/gatewayd-plugin-sdk).

## Documentation

The docs cover all aspects of using GatewayD. Some highlights include:

- [Getting Started](https://docs.gatewayd.io): install, run an instance of GatewayD and test it in action.
- [Configuration](https://docs.gatewayd.io/using-gatewayd/configuration): learn how you can configure GatewayD to behave as you like.
  - [Global Configuration](https://docs.gatewayd.io/using-gatewayd/configuration#global-configuration): control GatewayD's behavior.
  - [Plugins Configuration](https://docs.gatewayd.io/using-gatewayd/configuration#plugins-configuration): control GatewayD's plugin registry and the plugins' behavior.
- [Observability](https://docs.gatewayd.io/using-gatewayd/observability): logs, metrics, and traces are GatewayD's first-class citizens.
- [Using Plugins](https://docs.gatewayd.io/using-plugins/plugins): learn how to use plugins.
- [Developing Plugins](https://docs.gatewayd.io/developing-plugins/plugin-developers-guide): learn how to develop your own plugins.
- [gatewayd-plugin-cache](https://docs.gatewayd.io/plugins/gatewayd-plugin-cache): learn how to use the `gatewayd-plugin-cache` plugin for caching queries and their results.
- [gatewayd-plugin-js](https://docs.gatewayd.io/plugins/gatewayd-plugin-js): learn how to use the experimental `gatewayd-plugin-js` plugin for executing JavaScript code in GatewayD.

## Contributing

We welcome contributions from everyone.<!-- Please read our [contributing guide](https://gatewayd-io.github.io/CONTIBUTING.md) for more details.--> Just open an [issue](https://github.com/gatewayd-io/gatewayd/issues) or send us a [pull request](https://github.com/gatewayd-io/gatewayd/pulls).

## License

GatewayD is licensed under the [Affero General Public License v3.0](https://github.com/gatewayd-io/gatewayd/blob/main/LICENSE).
