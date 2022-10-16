module github.com/gatewayd-io/gatewayd

go 1.18

require (
	github.com/jackc/pgproto3 v1.1.0
	github.com/jackc/pgproto3/v2 v2.3.0
	github.com/panjf2000/gnet/v2 v2.1.2
	github.com/pganalyze/pg_query_go v1.0.3
)

require (
	github.com/jackc/chunkreader v1.0.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/sys v0.0.0-20221006211917-84dc82d7e875 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace github.com/auxten/postgresql-parser v1.0.0 => ../postgresql-parser
