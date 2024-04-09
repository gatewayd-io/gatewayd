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
	ErrServerListenFailed = NewGatewayDError(
		ErrCodeServerListenFailed, "couldn't listen on the server", nil)
	ErrSplitHostPortFailed = NewGatewayDError(
		ErrCodeSplitHostPortFailed, "failed to split host:port", nil)
	ErrAcceptFailed = NewGatewayDError(
		ErrCodeAcceptFailed, "failed to accept connection", nil)
	ErrGetTLSConfigFailed = NewGatewayDError(
		ErrCodeGetTLSConfigFailed, "failed to get TLS config", nil)
	ErrUpgradeToTLSFailed = NewGatewayDError(
		ErrCodeUpgradeToTLSFailed, "failed to upgrade to TLS", nil)

	ErrReadFailed = NewGatewayDError(
		ErrCodeReadFailed, "failed to read from the client", nil)

	ErrNilPointer = NewGatewayDError(
		ErrCodeNilPointer, "nil pointer", nil)

	ErrCastFailed = NewGatewayDError(
		ErrCodeCastFailed, "failed to cast", nil)

	ErrHookTerminatedConnection = NewGatewayDError(
		ErrCodeHookTerminatedConnection, "hook terminated connection", nil)

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
	ErrRunningActionTimeout = NewGatewayDError(
		ErrCodeRunError, "timeout running action", nil)
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
	FailedToCreateClient      = 1
	FailedToInitializePool    = 2
	FailedToStartServer       = 3
	FailedToStartTracer       = 4
	FailedToCreateActRegistry = 5
)
