package hook

const (
	// Run command hooks (cmd/run.go).
	OnConfigLoaded Type = "onConfigLoaded"
	OnNewLogger    Type = "onNewLogger"
	OnNewPool      Type = "onNewPool"
	OnNewProxy     Type = "onNewProxy"
	OnNewServer    Type = "onNewServer"
	OnSignal       Type = "onSignal"
	// Server hooks (network/server.go).
	OnRun               Type = "onRun"
	OnBooting           Type = "onBooting"
	OnBooted            Type = "onBooted"
	OnOpening           Type = "onOpening"
	OnOpened            Type = "onOpened"
	OnClosing           Type = "onClosing"
	OnClosed            Type = "onClosed"
	OnTraffic           Type = "onTraffic"
	OnTrafficFromClient Type = "onTrafficFromClient"
	OnTrafficToServer   Type = "onTrafficToServer"
	OnTrafficFromServer Type = "onTrafficFromServer"
	OnTrafficToClient   Type = "onTrafficToClient"
	OnShutdown          Type = "onShutdown"
	OnTick              Type = "onTick"
	// Pool hooks (network/pool.go).
	OnNewClient Type = "onNewClient"
)
