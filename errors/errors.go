package errors

import "errors"

const (
	ErrCodeUnknown ErrCode = iota
	ErrCodeNilContext
	ErrCodeClientNotFound
	ErrCodeClientNotConnected
	ErrCodeClientConnectionFailed
	ErrCodeNetworkNotSupported
	ErrCodeResolveFailed
	ErrCodePoolExhausted
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
	ErrCodeServerListenFailed
	ErrCodeSplitHostPortFailed
	ErrCodeAcceptFailed
	ErrCodeGetTLSConfigFailed
	ErrCodeTLSDisabled
	ErrCodeUpgradeToTLSFailed
	ErrCodeReadFailed
	ErrCodePutFailed
	ErrCodeNilPointer
	ErrCodeCastFailed
	ErrCodeHookReturnedError
	ErrCodeHookTerminatedConnection
	ErrCodeFileNotFound
	ErrCodeFileOpenFailed
	ErrCodeFileReadFailed
	ErrCodeDuplicateMetricsCollector
	ErrCodeInvalidMetricType
	ErrCodeValidationFailed
	ErrCodeLintingFailed
	ErrCodeExtractFailed
	ErrCodeDownloadFailed
	ErrCodeKeyNotFound
	ErrCodeRunError
	ErrCodeAsyncAction
	ErrCodeEvalError
	ErrCodeMsgEncodeError
	ErrCodePathSlipError
)

var (
	ErrClientNotFound = NewGatewayDError(
		ErrCodeClientNotFound, "client not found", nil)
	ErrNilContext = NewGatewayDError(
		ErrCodeNilContext, "context is nil", nil)
	ErrClientNotConnected = NewGatewayDError(
		ErrCodeClientNotConnected, "client is not connected", nil)
	ErrClientConnectionFailed = NewGatewayDError(
		ErrCodeClientConnectionFailed, "failed to create a new connection", nil)
	ErrNetworkNotSupported = NewGatewayDError(
		ErrCodeNetworkNotSupported, "network is not supported", nil)
	ErrResolveFailed = NewGatewayDError(
		ErrCodeResolveFailed, "failed to resolve address", nil)
	ErrPoolExhausted = NewGatewayDError(
		ErrCodePoolExhausted, "pool is exhausted", nil)

	ErrPluginNotFound = NewGatewayDError(
		ErrCodePluginNotFound, "plugin not found", nil)
	ErrPluginNotReady = NewGatewayDError(
		ErrCodePluginNotReady, "plugin is not ready", nil)
	ErrFailedToStartPlugin = NewGatewayDError(
		ErrCodeStartPluginFailed, "failed to start plugin", nil)
	ErrFailedToGetRPCClient = NewGatewayDError(
		ErrCodeGetRPCClientFailed, "failed to get RPC client", nil)
	ErrFailedToDispensePlugin = NewGatewayDError(
		ErrCodeDispensePluginFailed, "failed to dispense plugin", nil)
	ErrFailedToMergePluginMetrics = NewGatewayDError(
		ErrCodePluginMetricsMergeFailed, "failed to merge plugin metrics", nil)
	ErrFailedToPingPlugin = NewGatewayDError(
		ErrCodePluginPingFailed, "failed to ping plugin", nil)

	ErrClientReceiveFailed = NewGatewayDError(
		ErrCodeClientReceiveFailed, "couldn't receive data from the server", nil)
	ErrClientSendFailed = NewGatewayDError(
		ErrCodeClientSendFailed, "couldn't send data to the server", nil)

	ErrServerSendFailed = NewGatewayDError(
		ErrCodeServerSendFailed, "couldn't send data to the client", nil)
	ErrServerReceiveFailed = NewGatewayDError(
		ErrCodeServerReceiveFailed, "couldn't receive data from the client", nil)
	ErrServerListenFailed = NewGatewayDError(
		ErrCodeServerListenFailed, "couldn't listen on the server", nil)
	ErrSplitHostPortFailed = NewGatewayDError(
		ErrCodeSplitHostPortFailed, "failed to split host:port", nil)
	ErrAcceptFailed = NewGatewayDError(
		ErrCodeAcceptFailed, "failed to accept connection", nil)
	ErrGetTLSConfigFailed = NewGatewayDError(
		ErrCodeGetTLSConfigFailed, "failed to get TLS config", nil)
	ErrTLSDisabled = NewGatewayDError(
		ErrCodeTLSDisabled, "TLS is disabled or handshake failed", nil)
	ErrUpgradeToTLSFailed = NewGatewayDError(
		ErrCodeUpgradeToTLSFailed, "failed to upgrade to TLS", nil)

	ErrReadFailed = NewGatewayDError(
		ErrCodeReadFailed, "failed to read from the client", nil)

	ErrPutFailed = NewGatewayDError(
		ErrCodePutFailed, "failed to put in pool", nil)
	ErrNilPointer = NewGatewayDError(
		ErrCodeNilPointer, "nil pointer", nil)

	ErrCastFailed = NewGatewayDError(
		ErrCodeCastFailed, "failed to cast", nil)

	ErrHookReturnedError = NewGatewayDError(
		ErrCodeHookReturnedError, "hook returned error", nil)
	ErrHookTerminatedConnection = NewGatewayDError(
		ErrCodeHookTerminatedConnection, "hook terminated connection", nil)

	ErrFileNotFound = NewGatewayDError(
		ErrCodeFileNotFound, "file not found", nil)
	ErrFileOpenFailed = NewGatewayDError(
		ErrCodeFileOpenFailed, "failed to open file", nil)
	ErrFileReadFailed = NewGatewayDError(
		ErrCodeFileReadFailed, "failed to read file", nil)

	ErrDuplicateMetricsCollector = NewGatewayDError(
		ErrCodeDuplicateMetricsCollector, "duplicate metrics collector", nil)
	ErrInvalidMetricType = NewGatewayDError(
		ErrCodeInvalidMetricType, "invalid metric type", nil)

	ErrValidationFailed = NewGatewayDError(
		ErrCodeValidationFailed, "validation failed", nil)
	ErrLintingFailed = NewGatewayDError(
		ErrCodeLintingFailed, "linting failed", nil)

	ErrExtractFailed = NewGatewayDError(
		ErrCodeExtractFailed, "failed to extract the archive", nil)
	ErrDownloadFailed = NewGatewayDError(
		ErrCodeDownloadFailed, "failed to download the file", nil)

	ErrActionNotExist = NewGatewayDError(
		ErrCodeKeyNotFound, "action does not exist", nil)
	ErrRunningAction = NewGatewayDError(
		ErrCodeRunError, "error running action", nil)
	ErrAsyncAction = NewGatewayDError(
		ErrCodeAsyncAction, "async action", nil)
	ErrActionNotMatched = NewGatewayDError(
		ErrCodeKeyNotFound, "no matching action", nil)
	ErrPolicyNotMatched = NewGatewayDError(
		ErrCodeKeyNotFound, "no matching policy", nil)
	ErrEvalError = NewGatewayDError(
		ErrCodeEvalError, "error evaluating expression", nil)
	ErrMsgEncodeError = NewGatewayDError(
		ErrCodeMsgEncodeError, "error encoding message", nil)

	// Unwrapped errors.
	ErrLoggerRequired = errors.New("terminate action requires a logger parameter")
)

const (
	FailedToLoadPluginConfig  = 1
	FailedToLoadGlobalConfig  = 2
	FailedToCreateClient      = 3
	FailedToInitializePool    = 4
	FailedToStartServer       = 5
	FailedToStartTracer       = 6
	FailedToCreateActRegistry = 7
)
