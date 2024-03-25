package errors

const (
	ErrCodeUnknown ErrCode = iota
	ErrCodeNilContext
	ErrCodeClientNotFound
	ErrCodeClientNotConnected
	ErrCodeClientConnectionFailed
	ErrCodeNetworkNotSupported
	ErrCodeResolveFailed
	ErrCodePoolExhausted
	ErrCodeStartServerFailed
	ErrCodePluginNotFound
	ErrCodePluginNotReady
	ErrCodeStartPluginFailed
	ErrCodeGetRPCClientFailed
	ErrCodeDispensePluginFailed
	ErrCodePluginMetricsMergeFailed
	ErrCodePluginPingFailed
	ErrCodeClientReceiveFailed
	ErrCodeClientSendFailed
	ErrCodeServerReceiveFailed
	ErrCodeServerSendFailed
	ErrCodePutFailed
	ErrCodeCastFailed
	ErrCodeHookVerificationFailed
	ErrCodeHookReturnedError
	ErrCodeHookTerminatedConnection
	ErrCodeFileNotFound
	ErrCodeFileOpenFailed
	ErrCodeFileReadFailed
	ErrCodeDuplicateMetricsCollector
	ErrCodeInvalidMetricType
	ErrCodeValidationFailed
)

var (
	ErrClientNotFound = &GatewayDError{
		Code:          ErrCodeClientNotFound,
		Message:       "client not found",
		OriginalError: nil,
	}
	ErrNilContext = &GatewayDError{
		Code:          ErrCodeNilContext,
		Message:       "context is nil",
		OriginalError: nil,
	}
	ErrClientNotConnected = &GatewayDError{
		Code:          ErrCodeClientNotConnected,
		Message:       "client is not connected",
		OriginalError: nil,
	}
	ErrClientConnectionFailed = &GatewayDError{
		Code:          ErrCodeClientConnectionFailed,
		Message:       "failed to create a new connection",
		OriginalError: nil,
	}
	ErrNetworkNotSupported = &GatewayDError{
		Code:          ErrCodeNetworkNotSupported,
		Message:       "network is not supported",
		OriginalError: nil,
	}
	ErrResolveFailed = &GatewayDError{
		Code:          ErrCodeResolveFailed,
		Message:       "failed to resolve address",
		OriginalError: nil,
	}
	ErrPoolExhausted = &GatewayDError{
		Code:          ErrCodePoolExhausted,
		Message:       "pool is exhausted",
		OriginalError: nil,
	}
	ErrFailedToStartServer = &GatewayDError{
		Code:          ErrCodeStartServerFailed,
		Message:       "failed to start server",
		OriginalError: nil,
	}

	ErrPluginNotFound = &GatewayDError{
		Code:          ErrCodePluginNotFound,
		Message:       "plugin not found",
		OriginalError: nil,
	}
	ErrPluginNotReady = &GatewayDError{
		Code:          ErrCodePluginNotReady,
		Message:       "plugin is not ready",
		OriginalError: nil,
	}
	ErrFailedToStartPlugin = &GatewayDError{
		Code:          ErrCodeStartPluginFailed,
		Message:       "failed to start plugin",
		OriginalError: nil,
	}
	ErrFailedToGetRPCClient = &GatewayDError{
		Code:          ErrCodeGetRPCClientFailed,
		Message:       "failed to get RPC client",
		OriginalError: nil,
	}
	ErrFailedToDispensePlugin = &GatewayDError{
		Code:          ErrCodeDispensePluginFailed,
		Message:       "failed to dispense plugin",
		OriginalError: nil,
	}
	ErrFailedToMergePluginMetrics = &GatewayDError{
		Code:          ErrCodePluginMetricsMergeFailed,
		Message:       "failed to merge plugin metrics",
		OriginalError: nil,
	}
	ErrFailedToPingPlugin = &GatewayDError{
		Code:          ErrCodePluginPingFailed,
		Message:       "failed to ping plugin",
		OriginalError: nil,
	}

	ErrClientReceiveFailed = &GatewayDError{
		Code:          ErrCodeClientReceiveFailed,
		Message:       "couldn't receive data from the server",
		OriginalError: nil,
	}
	ErrClientSendFailed = &GatewayDError{
		Code:          ErrCodeClientSendFailed,
		Message:       "couldn't send data to the server",
		OriginalError: nil,
	}

	ErrServerSendFailed = &GatewayDError{
		Code:          ErrCodeServerSendFailed,
		Message:       "couldn't send data to the client",
		OriginalError: nil,
	}
	ErrServerReceiveFailed = &GatewayDError{
		Code:          ErrCodeServerReceiveFailed,
		Message:       "couldn't receive data from the client",
		OriginalError: nil,
	}

	ErrPutFailed = &GatewayDError{
		Code:          ErrCodePutFailed,
		Message:       "failed to put in pool",
		OriginalError: nil,
	}

	ErrCastFailed = &GatewayDError{
		Code:          ErrCodeCastFailed,
		Message:       "failed to cast",
		OriginalError: nil,
	}

	ErrHookVerificationFailed = &GatewayDError{
		Code:          ErrCodeHookVerificationFailed,
		Message:       "failed to verify hook",
		OriginalError: nil,
	}
	ErrHookReturnedError = &GatewayDError{
		Code:          ErrCodeHookReturnedError,
		Message:       "hook returned error",
		OriginalError: nil,
	}
	ErrHookTerminatedConnection = &GatewayDError{
		Code:          ErrCodeHookTerminatedConnection,
		Message:       "hook terminated connection",
		OriginalError: nil,
	}

	ErrFileNotFound = &GatewayDError{
		Code:          ErrCodeFileNotFound,
		Message:       "file not found",
		OriginalError: nil,
	}
	ErrFileOpenFailed = &GatewayDError{
		Code:          ErrCodeFileOpenFailed,
		Message:       "failed to open file",
		OriginalError: nil,
	}
	ErrFileReadFailed = &GatewayDError{
		Code:          ErrCodeFileReadFailed,
		Message:       "failed to read file",
		OriginalError: nil,
	}

	ErrDuplicateMetricsCollector = &GatewayDError{
		Code:          ErrCodeDuplicateMetricsCollector,
		Message:       "duplicate metrics collector",
		OriginalError: nil,
	}
	ErrInvalidMetricType = &GatewayDError{
		Code:          ErrCodeInvalidMetricType,
		Message:       "invalid metric type",
		OriginalError: nil,
	}

	ErrValidationFailed = &GatewayDError{
		Code:          ErrCodeValidationFailed,
		Message:       "validation failed",
		OriginalError: nil,
	}
)

const (
	FailedToLoadPluginConfig = 1
	FailedToLoadGlobalConfig = 2
	FailedToCreateClient     = 3
	FailedToInitializePool   = 4
	FailedToStartServer      = 5
)
