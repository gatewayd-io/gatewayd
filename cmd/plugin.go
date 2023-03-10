package cmd

import (
	"context"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/spf13/cobra"
)

// pluginCmd represents the plugin command.
var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Run: func(cmd *cobra.Command, args []string) {
		pluginCtx := context.Background()
		// Load global and plugin configuration.
		conf = config.NewConfig(pluginCtx, globalConfigFile, pluginConfigFile)
		conf.InitConfig(pluginCtx)

		// Create and initialize loggers from the config.
		for name, cfg := range conf.Global.Loggers {
			loggers[name] = logging.NewLogger(pluginCtx, logging.LoggerConfig{
				Output:            cfg.GetOutput(),
				Level:             cfg.GetLevel(),
				TimeFormat:        cfg.GetTimeFormat(),
				ConsoleTimeFormat: cfg.GetConsoleTimeFormat(),
				NoColor:           cfg.NoColor,
				FileName:          cfg.FileName,
				MaxSize:           cfg.MaxSize,
				MaxBackups:        cfg.MaxBackups,
				MaxAge:            cfg.MaxAge,
				Compress:          cfg.Compress,
				LocalTime:         cfg.LocalTime,
				SyslogPriority:    cfg.GetSyslogPriority(),
				RSyslogNetwork:    cfg.RSyslogNetwork,
				RSyslogAddress:    cfg.RSyslogAddress,
			})
		}

		// Set the default logger.
		logger := loggers[config.Default]

		logger.Debug().Msg("Adding a new plugin")
	},
}

func init() {
	rootCmd.AddCommand(pluginCmd)
}
