<p align="center">
    <a href="https://gatewayd.io/">
        <picture>
            <source media="(prefers-color-scheme: dark)" srcset="https://github.com/gatewayd-io/gatewayd/blob/main/assets/gatewayd-logotype-dark.png">
            <img alt="GatewayD logotype" src="https://github.com/gatewayd-io/gatewayd/blob/main/assets/gatewayd-logotype-light.png">
        </picture>
    </a>
</p>

GatewayD is a cloud-native database gateway and framework for building data-driven applications. It sits between your database(s) and your database client(s) and proxies all queries and their responses from the database.

## Architecture

The GatewayD architecture consists of a core, SDK and plugins. The core enables basic functionality for:

- **Proxying connections** between clients and server(s)
- **Connection pooling**, health check and management
- **Configuration management** for the core and plugins (YAML, env and runtime)
- **Logging** to console, stdout/stderr, file and (r)syslog
- **Plugin system and hooks**
- **Metric aggregation and emission**

![GatewayD Core Architecture v1](https://github.com/gatewayd-io/gatewayd/blob/main/assets/architecture-core-v1.png)

Then, plugins are loaded on startup to add tons of functionality, for example:

- **Query parsing and processing**
- **Caching**
- **Schema and data** management and transformation
- Many other possibilities

![GatewayD Core Architecture v1](https://github.com/gatewayd-io/gatewayd/blob/main/assets/architecture-plugins-v1.png)

Plugins talk over **gRPC** using **protocol buffers** with the core. The core exposes a long list of hooks. Upon loading a plugin, the plugin can register to those hooks. When specific events happen in the core, like `onTrafficFromClient`, the plugins registered to that hook will be called with the parameters available in that hook, like the client request, that is, the query. Plugins can terminate client connections and return a response immediately without consulting the database server. Plugins can also emit Prometheus metrics via HTTP over UDS to the core. Then, the core aggregates, relabels and emits those metrics over an HTTP endpoint to be scraped by Prometheus.

The last piece of the puzzle is the SDK, which helps developers create their extensions with ease.

![GatewayD Core Architecture v1](https://github.com/gatewayd-io/gatewayd/blob/main/assets/architecture-sdk-v1.png)

## Run

To run GatewayD, you need to download the latest version from the [releases](https://github.com/gatewayd-io/gatewayd/releases) page. Then extract it somewhere in your `PATH` and run it like below:

```bash
# Run PostgreSQL in the background via Docker
$ docker run --rm --name postgres-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres
# Run Redis if you want to cache queries and their results (optional)
$ ./redis-server &
# Run GatewayD
$ ./gatewayd run
2023-01-24T22:57:27+01:00 WRN plugin configured with a nil SecureConfig
2023-01-24T22:57:27+01:00 INF configuring client automatic mTLS
2023-01-24T22:57:27+01:00 INF Starting metrics server via HTTP over Unix domain socket endpoint=/metrics timestamp=2023-01-24T22:57:27.848+0100 unixDomainSocket=/tmp/gatewayd-plugin-cache.sock
2023-01-24T22:57:27+01:00 INF configuring server automatic mTLS  timestamp=2023-01-24T22:57:27.849+0100
2023-01-24T22:57:27+01:00 INF Plugin is ready name=gatewayd-plugin-cache
2023-01-24T22:57:27+01:00 INF Started the metrics merger scheduler metricsMergerPeriod=5s startDelay=1674597452
2023-01-24T22:57:27+01:00 INF Metrics are exposed address=http://localhost:2112/metrics
2023-01-24T22:57:27+01:00 INF There are clients available in the pool count=10
2023-01-24T22:57:27+01:00 INF Started the client health check scheduler healthCheckPeriod=1m0s startDelay=1674597507
2023-01-24T22:57:27+01:00 INF GatewayD is listening address=0.0.0.0:15432
2023-01-24T22:57:27+01:00 INF GatewayD is running pid=4823
```

As shown above in the console logs, the `gatewayd-plugin-cache` loads and exposes metrics over HTTP via `/tmp/gatewayd-plugin-cache.sock`. The metrics merger is started and collects, aggregates and relabels metrics from plugins, and merges them with metrics emitted from the core. It'll then expose those metrics over HTTP via `http://localhost:2112/metrics`. Ten connections are connected to the PostgreSQL on port 5432 and are put in the pool, ready to serve incoming connections from clients. Next, a connection health check is run to recycle connections when no authenticated client exists - This is to deal with timeout on the database server. When all the above is set up, GatewayD starts listening on port 15432. The clients can point to GatewayD's address and start working as before while GatewayD is proxying calls. If you run Redis, `SELECT` queries sent through GatewayD will end up being cached along with their response from PostgreSQL. The next time another client runs the same query, the response will be read from the cache and returned to the client, without touching server. When done testing, you should be able to gracefully stop GatewayD using `CTRL+C`. In the above example, only one plugin, `gatewayd-plugin-cache`, is loaded, but GatewayD is perfectly capable of handling multiple plugins.
