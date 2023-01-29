package errors

const (
	ErrCodeUnknown ErrCode = iota
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
	ErrCodeClientReceiveFailed
	ErrCodeClientSendFailed
	ErrCodeServerReceiveFailed
	ErrCodeServerSendFailed
	ErrCodePutFailed
	ErrCodeCastFailed
	ErrCodeHookVerificationFailed
	ErrCodeHookReturnedError
	ErrCodeFileNotFound
	ErrCodeFileOpenFailed
	ErrCodeFileReadFailed
)

var (
	ErrClientNotFound = NewGatewayDError(
		ErrCodeClientNotFound, "client not found", nil)
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
	ErrFailedToStartServer = NewGatewayDError(
		ErrCodeStartServerFailed, "failed to start server", nil)

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

	ErrClientReceiveFailed = NewGatewayDError(
		ErrCodeClientReceiveFailed, "couldn't receive data from the server", nil)
	ErrClientSendFailed = NewGatewayDError(
		ErrCodeClientSendFailed, "couldn't send data to the server", nil)

	ErrServerSendFailed = NewGatewayDError(
		ErrCodeServerSendFailed, "couldn't send data to the client", nil)
	ErrServerReceiveFailed = NewGatewayDError(
		ErrCodeServerReceiveFailed, "couldn't receive data from the client", nil)

	ErrPutFailed = NewGatewayDError(
		ErrCodePutFailed, "failed to put in pool", nil)

	ErrCastFailed = NewGatewayDError(
		ErrCodeCastFailed, "failed to cast", nil)

	ErrHookVerificationFailed = NewGatewayDError(
		ErrCodeHookVerificationFailed, "failed to verify hook", nil)
	ErrHookReturnedError = NewGatewayDError(
		ErrCodeHookReturnedError, "hook returned error", nil)

	ErrFileNotFound = NewGatewayDError(
		ErrCodeFileNotFound, "file not found", nil)
	ErrFileOpenFailed = NewGatewayDError(
		ErrCodeFileOpenFailed, "failed to open file", nil)
	ErrFileReadFailed = NewGatewayDError(
		ErrCodeFileReadFailed, "failed to read file", nil)
)

const (
	FailedToLoadPluginConfig = 1
	FailedToLoadGlobalConfig = 2
)
