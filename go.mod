module github.com/gatewayd-io/gatewayd

go 1.19

require (
	github.com/Masterminds/semver/v3 v3.2.0
	github.com/fergusstrange/embedded-postgres v1.19.0
	github.com/gatewayd-io/gatewayd-plugin-test v0.0.0-00010101000000-000000000000
	github.com/google/go-cmp v0.5.9
	github.com/hashicorp/go-plugin v1.4.8
	github.com/knadh/koanf v1.4.4
	github.com/panjf2000/gnet/v2 v2.2.0
	github.com/rs/zerolog v1.28.0
	github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify v1.8.1
	google.golang.org/grpc v1.51.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/go-hclog v1.4.0 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lib/pq v1.10.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/net v0.4.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/text v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20221207170731-23e4bf6bdc37 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gatewayd-io/gatewayd-plugin-test => ../gatewayd-plugin-test
