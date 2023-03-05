<p align="center">
    <a href="https://gatewayd.io/">
        <picture>
            <source media="(prefers-color-scheme: dark)" srcset="https://github.com/gatewayd-io/gatewayd/blob/main/assets/gatewayd-logotype-dark.png">
            <img alt="GatewayD logotype" src="https://github.com/gatewayd-io/gatewayd/blob/main/assets/gatewayd-logotype-light.png">
        </picture>
    </a>
</p>

GatewayD is a cloud-native database gateway and framework for building data-driven applications. It sits between your database servers and clients and proxies all their communication.

## Architecture

The GatewayD architecture consists of a core, SDK and plugins. The core enables these functionalities:

- **Event-based I/O** for handling connections:

    Many events happen upon boot, running servers and proxying connections. Each event registers one or more hooks, so plugins can connect to and receive the event and its data. Plugins can alter and return what they receive. For example, there is a hook called `onTrafficFromClient`. Plugins registered to this hook can see what the client sends and manipulate it before sending it to the server. For example, the result of an SQL query can be cached in Redis (or any other caching server) using the query as the key and the database's response as the value. Any other client that sends the same query will receive the cached results without reaching the database server, until one updates the record or the cache key expires. Of course, the query's result is cached in the `onTrafficFromServer` hook.

- **Connection pooling**, health check and management:

    There are two pools for handling incoming client connections. Upon startup, the server initiates a fixed number of connections to the database servers and puts them in the available connections pool. Whenever a new client wants to connect, it'll grab an available connection and puts it into the busy connection pool, thus effectively mapping these two connections together for proxying. Stale and closed connections will be recycled.

- **Connection proxy** between clients and server(s):

    Client connections in the busy connections pool listen to specific events: `onConnect`, `onClose`, `onTraffic`, and so on. When traffic is seen on a client connection, the proxy relays information between the client and the server. While proxying, four important hooks are called: `onTrafficFromClient`, `onTrafficToServer`, `onTrafficFromServer` and `onTrafficToClient`. These hooks pass any data they receive to plugins and plugins get to decide what to do with the data based on their priority. Multiple plugins can register to a single hook.

- **Configuration management** for the core and plugins (file-based, env-vars and runtime):

    There are two primary configurations: 1) global configuration, which is managed by the `gatewayd.yaml` file. 2) plugins configuration is managed by the `gatewayd_plugins.yaml` file. The global configuration includes directives for configuring loggers, metrics, clients, pools, proxies and servers. The plugins configuration includes directives for managing the plugin registry and its subsystems plus the actual plugins configuration. Both of these can be overridden using environment variables. Plugins can also override the global configuration.

- **Multi-tenancy** for supporting multiple databases and clients:

    Multiple databases can be connected to the core and clients can choose which database to connect. The work is still in progress for aggregation between databases.

- **Full observability** including logging, metrics and tracing:

    Observability is a first-class citizen of GatewayD. Whatever happens in the core and the plugins should be observable.

- **Logging** to console, stdout/stderr, file and (r)syslog:

    The core can write logs to multiple log outputs. The log level controls how much information is written to the logs. Detailed information is written as traces and the rest have higher log levels. Log rotation and compression are also supported if logs are written to files.

- **Metric merger and emission**:

    The core exposes Prometheus metrics over HTTP. In turn, plugins can either expose their metrics or use the SDK to expose metrics over HTTP via a Unix Domain Socket. The UDS addresses get registered in the metrics merger. The metrics merger is an in-process scheduler that runs every few seconds and collects metrics from plugins. Collected metrics are relabeled and merged with the core metrics and are exposed over HTTP via TCP.

- **Tracing** to observe deep into the core:

    The core produces traces in the OpenTelemetry gRPC format. The trace exporter can connect with any tracing framework supporting OpenTelemetry over gRPC, such as Jaeger. Many pieces of information are recorded in traces and can be correlated with logs and metrics for an excellent observability experience.

- **Plugin registry** for plugin loading and management:

    The plugin registry controls the entire lifecycle of plugins, including loading them and registering their hooks in the hooks registry. It also controls the running of gRPC endpoints exposed by plugins registered with each hook. Plugins are always loaded by the core and communicate with the core over gRPC.

- **Plugin hooks** for tapping into the event-based I/O:

    The core executes many events in the entire lifetime of a pair of connections and other parts of the system. Plugins expose gRPC endpoints. Each endpoint can register to a hook. Since multiple plugins can be loaded, each can register to one or many hooks. The order of appearance in the plugins configurations dictates which priority each plugin has. Each hook is a binding between an event in the core and a corresponding gRPC endpoint in one or more plugin(s).

- **Plugin health check**:

    The metrics merger and the plugin registry rely on plugins always being healthy. Plugin crashes can cause malfunctions in the core and the metrics merger. The health check process pings plugins every few seconds and removes the faulty ones from the register and the metrics merger scheduler. The crashed plugins will be automatically reloaded.

- **Admin API**

    The current API exposes a few endpoints to retrieve information about various parts of the system over gRPC and HTTP. The HTTP API server also serves a `swagger.json` that contains all the endpoints and their examples. The Swagger UI is also accessible that provides a UI on top of the `swagger.json` specification document.

<p align="center">
    <img alt="GatewayD Core Architecture v1" src="https://github.com/gatewayd-io/gatewayd/blob/main/assets/architecture-core-v1.png" style="width: 512px;">
</p>

Then, the plugins are loaded on startup to add tons of functionality, for example:

- **Query parsing and processing**
- **Caching**
- **Schema and data** management and transformation
- Many other possibilities

<p align="center">
    <img alt="GatewayD Plugins Architecture v1" src="https://github.com/gatewayd-io/gatewayd/blob/main/assets/architecture-plugins-v1.png" style="width: 512px;">
</p>

Plugins talk over **gRPC** using **protocol buffers** with the core. The core exposes a long list of hooks. Upon loading a plugin, the plugin can register to those hooks. When specific events happen in the core, like `onTrafficFromClient`, the plugins registered to that hook will be called with the parameters available: the client request containing the query. Plugins can terminate client connections and return a response immediately without consulting the database server. Plugins can also emit Prometheus metrics via HTTP over UDS to the core. Then, the core aggregates, relabels and emits those metrics over an HTTP endpoint to be scraped by Prometheus.

The last piece of the puzzle is the SDK, which helps developers create extensions with ease.

<p align="center">
    <img alt="GatewayD SDK Architecture v1" src="https://github.com/gatewayd-io/gatewayd/blob/main/assets/architecture-sdk-v1.png" style="width: 512px;">
</p>

## Run

To run GatewayD, you need to download the latest version from the [releases](https://github.com/gatewayd-io/gatewayd/releases) page. Then extract it somewhere in your `PATH` and run it like below:

```bash
# Run PostgreSQL in the background via Docker
$ docker run --rm --name postgres-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres
# Run Redis if you want to cache queries and their results (optional)
$ ./redis-server &
# Run GatewayD
$ ./gatewayd run
2023-02-06T 20:20:32+01:00 WRN plugin configured with a nil SecureConfig plugin=gatewayd-plugin-cache
2023-02-06T 20:20:32+01:00 INF configuring client automatic mTLS plugin=gatewayd-plugin-cache
2023-02-06T 20:20:32+01:00 INF Starting metrics server via HTTP over Unix domain socket endpoint=/metrics plugin=gatewayd-plugin-cache timestamp=2023-02-06T 20:20:32.170+0100 unixDomainSocket=/tmp/gatewayd-plugin-cache.sock
2023-02-06T 20:20:32+01:00 INF configuring server automatic mTLS plugin=gatewayd-plugin-cache timestamp=2023-02-06T 20:20:32.170+0100
2023-02-06T 20:20:32+01:00 INF Plugin is ready name=gatewayd-plugin-cache
2023-02-06T 20:20:32+01:00 INF Started the metrics merger scheduler metricsMergerPeriod=5s startDelay=1675639237
2023-02-06T 20:20:32+01:00 INF Starting plugin health check scheduler healthCheckPeriod=5s
2023-02-06T 20:20:32+01:00 INF Metrics are exposed address=http://localhost:2112/metrics
2023-02-06T 20:20:32+01:00 INF There are clients available in the pool count=10 name=default
2023-02-06T 20:20:32+01:00 INF Started the client health check scheduler healthCheckPeriod=1m0s startDelay=1675639292
2023-02-06T 20:20:32+01:00 INF GatewayD is listening address=0.0.0.0:15432
2023-02-06T 20:20:32+01:00 INF GatewayD is running pid=9566
```

As shown above in the console logs, the `gatewayd-plugin-cache` loads and exposes metrics over HTTP via `/tmp/gatewayd-plugin-cache.sock`. The metrics merger is started and collects, aggregates and relabels metrics from plugins, and merges them with metrics emitted from the core. It'll then expose those metrics over HTTP via `http://localhost:2112/metrics`. Ten connections are connected to the PostgreSQL on port 5432 and are put in the pool, ready to serve incoming connections from clients. Next, a connection health check is run to recycle connections when no authenticated client exists - This is to deal with timeout on the database server. When all the above is set up, GatewayD starts listening on port 15432. The clients can point to GatewayD's address and start working as before while GatewayD is proxying calls. If you run Redis, `SELECT` queries sent through GatewayD and their response from PostgreSQL are cached. The next time another client runs the same query, the response will be read from the cache and returned to the client, without touching the server. When done testing, you should be able to gracefully stop GatewayD using `CTRL+C`. In the above example, only one plugin, `gatewayd-plugin-cache`, is loaded, but GatewayD is perfectly capable of handling multiple plugins.
