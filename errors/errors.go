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
	ErrClientNotFound = NewGatewayDError(GatewayDError{
		Code:          ErrCodeClientNotFound,
		Message:       "client not found",
		OriginalError: nil,
	})
	ErrNilContext = NewGatewayDError(GatewayDError{
		Code:          ErrCodeNilContext,
		Message:       "context is nil",
		OriginalError: nil,
	})
	ErrClientNotConnected = NewGatewayDError(GatewayDError{
		Code:          ErrCodeClientNotConnected,
		Message:       "client is not connected",
		OriginalError: nil,
	})
	ErrClientConnectionFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeClientConnectionFailed,
		Message:       "failed to create a new connection",
		OriginalError: nil,
	})
	ErrNetworkNotSupported = NewGatewayDError(GatewayDError{
		Code:          ErrCodeNetworkNotSupported,
		Message:       "network is not supported",
		OriginalError: nil,
	})
	ErrResolveFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeResolveFailed,
		Message:       "failed to resolve address",
		OriginalError: nil,
	})
	ErrPoolExhausted = NewGatewayDError(GatewayDError{
		Code:          ErrCodePoolExhausted,
		Message:       "pool is exhausted",
		OriginalError: nil,
	})
	ErrFailedToStartServer = NewGatewayDError(GatewayDError{
		Code:          ErrCodeStartServerFailed,
		Message:       "failed to start server",
		OriginalError: nil,
	})

	ErrPluginNotFound = NewGatewayDError(GatewayDError{
		Code:          ErrCodePluginNotFound,
		Message:       "plugin not found",
		OriginalError: nil,
	})
	ErrPluginNotReady = NewGatewayDError(GatewayDError{
		Code:          ErrCodePluginNotReady,
		Message:       "plugin is not ready",
		OriginalError: nil,
	})
	ErrFailedToStartPlugin = NewGatewayDError(GatewayDError{
		Code:          ErrCodeStartPluginFailed,
		Message:       "failed to start plugin",
		OriginalError: nil,
	})
	ErrFailedToGetRPCClient = NewGatewayDError(GatewayDError{
		Code:          ErrCodeGetRPCClientFailed,
		Message:       "failed to get RPC client",
		OriginalError: nil,
	})
	ErrFailedToDispensePlugin = NewGatewayDError(GatewayDError{
		Code:          ErrCodeDispensePluginFailed,
		Message:       "failed to dispense plugin",
		OriginalError: nil,
	})
	ErrFailedToMergePluginMetrics = NewGatewayDError(GatewayDError{
		Code:          ErrCodePluginMetricsMergeFailed,
		Message:       "failed to merge plugin metrics",
		OriginalError: nil,
	})
	ErrFailedToPingPlugin = NewGatewayDError(GatewayDError{
		Code:          ErrCodePluginPingFailed,
		Message:       "failed to ping plugin",
		OriginalError: nil,
	})

	ErrClientReceiveFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeClientReceiveFailed,
		Message:       "couldn't receive data from the server",
		OriginalError: nil,
	})
	ErrClientSendFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeClientSendFailed,
		Message:       "couldn't send data to the server",
		OriginalError: nil,
	})

	ErrServerSendFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeServerSendFailed,
		Message:       "couldn't send data to the client",
		OriginalError: nil,
	})
	ErrServerReceiveFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeServerReceiveFailed,
		Message:       "couldn't receive data from the client",
		OriginalError: nil,
	})

	ErrPutFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodePutFailed,
		Message:       "failed to put in pool",
		OriginalError: nil,
	})

	ErrCastFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeCastFailed,
		Message:       "failed to cast",
		OriginalError: nil,
	})

	ErrHookVerificationFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeHookVerificationFailed,
		Message:       "failed to verify hook",
		OriginalError: nil,
	})
	ErrHookReturnedError = NewGatewayDError(GatewayDError{
		Code:          ErrCodeHookReturnedError,
		Message:       "hook returned error",
		OriginalError: nil,
	})
	ErrHookTerminatedConnection = NewGatewayDError(GatewayDError{
		Code:          ErrCodeHookTerminatedConnection,
		Message:       "hook terminated connection",
		OriginalError: nil,
	})

	ErrFileNotFound = NewGatewayDError(GatewayDError{
		Code:          ErrCodeFileNotFound,
		Message:       "file not found",
		OriginalError: nil,
	})
	ErrFileOpenFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeFileOpenFailed,
		Message:       "failed to open file",
		OriginalError: nil,
	})
	ErrFileReadFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeFileReadFailed,
		Message:       "failed to read file",
		OriginalError: nil,
	})

	ErrDuplicateMetricsCollector = NewGatewayDError(GatewayDError{
		Code:          ErrCodeDuplicateMetricsCollector,
		Message:       "duplicate metrics collector",
		OriginalError: nil,
	})
	ErrInvalidMetricType = NewGatewayDError(GatewayDError{
		Code:          ErrCodeInvalidMetricType,
		Message:       "invalid metric type",
		OriginalError: nil,
	})

	ErrValidationFailed = NewGatewayDError(GatewayDError{
		Code:          ErrCodeValidationFailed,
		Message:       "validation failed",
		OriginalError: nil,
	})
)

const (
	FailedToLoadPluginConfig = 1
	FailedToLoadGlobalConfig = 2
	FailedToCreateClient     = 3
	FailedToInitializePool   = 4
	FailedToStartServer      = 5
)
