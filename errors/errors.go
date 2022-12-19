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
	ErrCodeFileNotFound
	ErrCodeFileOpenFailed
	ErrCodeFileReadFailed
)

var (
	ErrClientNotFound = NewGatewaydError(
		ErrCodeClientNotFound, "client not found", nil)
	ErrClientNotConnected = NewGatewaydError(
		ErrCodeClientNotConnected, "client is not connected", nil)
	ErrClientConnectionFailed = NewGatewaydError(
		ErrCodeClientConnectionFailed, "failed to create a new connection", nil)
	ErrNetworkNotSupported = NewGatewaydError(
		ErrCodeNetworkNotSupported, "network is not supported", nil)
	ErrResolveFailed = NewGatewaydError(
		ErrCodeResolveFailed, "failed to resolve address", nil)
	ErrPoolExhausted = NewGatewaydError(
		ErrCodePoolExhausted, "pool is exhausted", nil)
	ErrFailedToStartServer = NewGatewaydError(
		ErrCodeStartServerFailed, "failed to start server", nil)

	ErrPluginNotFound = NewGatewaydError(
		ErrCodePluginNotFound, "plugin not found", nil)
	ErrPluginNotReady = NewGatewaydError(
		ErrCodePluginNotReady, "plugin is not ready", nil)
	ErrFailedToStartPlugin = NewGatewaydError(
		ErrCodeStartPluginFailed, "failed to start plugin", nil)
	ErrFailedToGetRPCClient = NewGatewaydError(
		ErrCodeGetRPCClientFailed, "failed to get RPC client", nil)
	ErrFailedToDispensePlugin = NewGatewaydError(
		ErrCodeDispensePluginFailed, "failed to dispense plugin", nil)

	ErrClientReceiveFailed = NewGatewaydError(
		ErrCodeClientReceiveFailed, "couldn't receive data from the server", nil)
	ErrClientSendFailed = NewGatewaydError(
		ErrCodeClientSendFailed, "couldn't send data to the server", nil)

	ErrServerSendFailed = NewGatewaydError(
		ErrCodeServerSendFailed, "couldn't send data to the client", nil)
	ErrServerReceiveFailed = NewGatewaydError(
		ErrCodeServerReceiveFailed, "couldn't receive data from the client", nil)

	ErrPutFailed = NewGatewaydError(
		ErrCodePutFailed, "failed to put in pool", nil)

	ErrCastFailed = NewGatewaydError(
		ErrCodeCastFailed, "failed to cast", nil)

	ErrHookVerificationFailed = NewGatewaydError(
		ErrCodeHookVerificationFailed, "failed to verify hook", nil)

	ErrFileNotFound = NewGatewaydError(
		ErrCodeFileNotFound, "file not found", nil)
	ErrFileOpenFailed = NewGatewaydError(
		ErrCodeFileOpenFailed, "failed to open file", nil)
	ErrFileReadFailed = NewGatewaydError(
		ErrCodeFileReadFailed, "failed to read file", nil)
)

const (
	FailedToLoadPluginConfig = 1
	FailedToLoadGlobalConfig = 2
)
