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
	ErrClientNotFound = &GatewayDError{
		ErrCodeClientNotFound, "client not found", nil}
	ErrNilContext = &GatewayDError{
		ErrCodeNilContext, "context is nil", nil}
	ErrClientNotConnected = &GatewayDError{
		ErrCodeClientNotConnected, "client is not connected", nil}
	ErrClientConnectionFailed = &GatewayDError{
		ErrCodeClientConnectionFailed, "failed to create a new connection", nil}
	ErrNetworkNotSupported = &GatewayDError{
		ErrCodeNetworkNotSupported, "network is not supported", nil}
	ErrResolveFailed = &GatewayDError{
		ErrCodeResolveFailed, "failed to resolve address", nil}
	ErrPoolExhausted = &GatewayDError{
		ErrCodePoolExhausted, "pool is exhausted", nil}

	ErrPluginNotFound = &GatewayDError{
		ErrCodePluginNotFound, "plugin not found", nil}
	ErrPluginNotReady = &GatewayDError{
		ErrCodePluginNotReady, "plugin is not ready", nil}
	ErrFailedToStartPlugin = &GatewayDError{
		ErrCodeStartPluginFailed, "failed to start plugin", nil}
	ErrFailedToGetRPCClient = &GatewayDError{
		ErrCodeGetRPCClientFailed, "failed to get RPC client", nil}
	ErrFailedToDispensePlugin = &GatewayDError{
		ErrCodeDispensePluginFailed, "failed to dispense plugin", nil}
	ErrFailedToMergePluginMetrics = &GatewayDError{
		ErrCodePluginMetricsMergeFailed, "failed to merge plugin metrics", nil}
	ErrFailedToPingPlugin = &GatewayDError{
		ErrCodePluginPingFailed, "failed to ping plugin", nil}

	ErrClientReceiveFailed = &GatewayDError{
		ErrCodeClientReceiveFailed, "couldn't receive data from the server", nil}
	ErrClientSendFailed = &GatewayDError{
		ErrCodeClientSendFailed, "couldn't send data to the server", nil}

	ErrServerSendFailed = &GatewayDError{
		ErrCodeServerSendFailed, "couldn't send data to the client", nil}
	ErrServerReceiveFailed = &GatewayDError{
		ErrCodeServerReceiveFailed, "couldn't receive data from the client", nil}
	ErrServerListenFailed = &GatewayDError{
		ErrCodeServerListenFailed, "couldn't listen on the server", nil}
	ErrSplitHostPortFailed = &GatewayDError{
		ErrCodeSplitHostPortFailed, "failed to split host:port", nil}
	ErrAcceptFailed = &GatewayDError{
		ErrCodeAcceptFailed, "failed to accept connection", nil}
	ErrGetTLSConfigFailed = &GatewayDError{
		ErrCodeGetTLSConfigFailed, "failed to get TLS config", nil}
	ErrTLSDisabled = &GatewayDError{
		ErrCodeTLSDisabled, "TLS is disabled or handshake failed", nil}
	ErrUpgradeToTLSFailed = &GatewayDError{
		ErrCodeUpgradeToTLSFailed, "failed to upgrade to TLS", nil}

	ErrReadFailed = &GatewayDError{
		ErrCodeReadFailed, "failed to read from the client", nil}

	ErrPutFailed = &GatewayDError{
		ErrCodePutFailed, "failed to put in pool", nil}
	ErrNilPointer = &GatewayDError{
		ErrCodeNilPointer, "nil pointer", nil}

	ErrCastFailed = &GatewayDError{
		ErrCodeCastFailed, "failed to cast", nil}

	ErrHookReturnedError = &GatewayDError{
		ErrCodeHookReturnedError, "hook returned error", nil}
	ErrHookTerminatedConnection = &GatewayDError{
		ErrCodeHookTerminatedConnection, "hook terminated connection", nil}

	ErrFileNotFound = &GatewayDError{
		ErrCodeFileNotFound, "file not found", nil}
	ErrFileOpenFailed = &GatewayDError{
		ErrCodeFileOpenFailed, "failed to open file", nil}
	ErrFileReadFailed = &GatewayDError{
		ErrCodeFileReadFailed, "failed to read file", nil}

	ErrDuplicateMetricsCollector = &GatewayDError{
		ErrCodeDuplicateMetricsCollector, "duplicate metrics collector", nil}
	ErrInvalidMetricType = &GatewayDError{
		ErrCodeInvalidMetricType, "invalid metric type", nil}

	ErrValidationFailed = &GatewayDError{
		ErrCodeValidationFailed, "validation failed", nil}
	ErrLintingFailed = &GatewayDError{
		ErrCodeLintingFailed, "linting failed", nil}

	ErrExtractFailed = &GatewayDError{
		ErrCodeExtractFailed, "failed to extract the archive", nil}
	ErrDownloadFailed = &GatewayDError{
		ErrCodeDownloadFailed, "failed to download the file", nil}

	ErrActionNotExist = &GatewayDError{
		ErrCodeKeyNotFound, "action does not exist", nil}
	ErrRunningAction = &GatewayDError{
		ErrCodeRunError, "error running action", nil}
	ErrAsyncAction = &GatewayDError{
		ErrCodeAsyncAction, "async action", nil}
	ErrRunningActionTimeout = &GatewayDError{
		ErrCodeRunError, "timeout running action", nil}
	ErrActionNotMatched = &GatewayDError{
		ErrCodeKeyNotFound, "no matching action", nil}
	ErrPolicyNotMatched = &GatewayDError{
		ErrCodeKeyNotFound, "no matching policy", nil}
	ErrEvalError = &GatewayDError{
		ErrCodeEvalError, "error evaluating expression", nil}
	ErrMsgEncodeError = &GatewayDError{
		ErrCodeMsgEncodeError, "error encoding message", nil}

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
