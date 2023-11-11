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
    <a href="https://coveralls.io/github/gatewayd-io/gatewayd?branch=main"><img src="https://coveralls.io/repos/github/gatewayd-io/gatewayd/badge.svg?branch=main" alt="Coverage Status" /></a>
</p>

**GatewayD** is a free and open-source cloud-native database gateway and framework for building data-driven applications. It sits between your database servers and clients and proxies all their communication. It is like API gateways, for databases.

GatewayD is an L4 proxy for SQL databases and clients. You can either write your own plugins or make use of our built-in, community and enterprise plugins.

Using GatewayD, you can see through the queries and the data passing between your database server and clients, and take action. For example, you can cache the result of SQL SELECT queries or detect and prevent SQL injection attacks.

GatewayD is developed by [GatewayD Labs](https://gatewayd.io/) and the community.

## Features

- **Cloud-native**:

    GatewayD is built with cloud-native principles in mind. It is containerized, <!--stateless, scalable,--> observable and extensible.

- **Connection pooling and proxying**:

    GatewayD pools connections to database servers and proxies them to clients. It also pools connections to clients and proxies them to database servers.

- **Database-agnostic**:

    GatewayD is database-agnostic. It supports databases through plugins.

- **Plugin-based & extensible**:

    GatewayD is plugin-based. You can write your own plugins or use our built-in, community and enterprise plugins.

- **Multi-tenancy**:

    GatewayD supports multiple databases and clients.

- **Full observability**:

    GatewayD is fully observable. It supports logging, metrics and tracing.

## Plugins & SDK

The plugins are the lifeblood of GatewayD. They are loaded on startup to add tons of functionality, for example:

- **Query parsing and processing**
- **Caching**
- **Injection detection and prevention**
- **Schema and data** management and transformation
- Many other possibilities

The plugins are *usually* written in Go and are compiled into stand-alone executables. They are loaded on startup and are ready to be used by the users. In turn, the developers can write their own plugins and use them in their applications, which is made straightforward by using the [SDK](https://github.com/gatewayd-io/gatewayd-plugin-sdk).

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

## Contributing

We welcome contributions from everyone.<!-- Please read our [contributing guide](https://gatewayd-io.github.io/CONTIBUTING.md) for more details.--> Just open an [issue](https://github.com/gatewayd-io/gatewayd/issues) or send us a [pull request](https://github.com/gatewayd-io/gatewayd/pulls).

## License

GatewayD is licensed under the [Affero General Public License v3.0](https://github.com/gatewayd-io/gatewayd/blob/main/LICENSE).
