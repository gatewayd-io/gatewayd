package network

import (
	"os"

	"github.com/knadh/koanf"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

type (
	Prio     uint
	HookType string
)

type HookDef func(...interface{})

const (
	OnConfigLoaded    HookType = "onConfigLoaded"
	OnNewLogger       HookType = "onNewLogger"
	OnNewPool         HookType = "onNewPool"
	OnNewProxy        HookType = "onNewProxy"
	OnNewServer       HookType = "onNewServer"
	OnSignal          HookType = "onSignal"
	OnRun             HookType = "onRun"
	OnBooting         HookType = "onBooting"
	OnBooted          HookType = "onBooted"
	OnOpening         HookType = "onOpening"
	OnOpened          HookType = "onOpened"
	OnClosing         HookType = "onClosing"
	OnClosed          HookType = "onClosed"
	OnTraffic         HookType = "onTraffic"
	OnIncomingTraffic HookType = "onIncomingTraffic"
	OnOutgoingTraffic HookType = "onOutgoingTraffic"
	OnShutdown        HookType = "onShutdown"
	OnTick            HookType = "onTick"
	OnNewClient       HookType = "onNewClient"
)

type HookConfig struct {
	onConfigLoaded map[Prio]HookDef
	onNewLogger    map[Prio]HookDef
	onNewPool      map[Prio]HookDef
	onNewProxy     map[Prio]HookDef
	onNewServer    map[Prio]HookDef
	onSignal       map[Prio]HookDef

	onRun map[Prio]HookDef

	onBooting map[Prio]HookDef
	onBooted  map[Prio]HookDef

	onOpening map[Prio]HookDef
	onOpened  map[Prio]HookDef

	onClosing map[Prio]HookDef
	onClosed  map[Prio]HookDef

	onTraffic         map[Prio]HookDef
	onIncomingTraffic map[Prio]Traffic
	onOutgoingTraffic map[Prio]Traffic

	onShutdown map[Prio]HookDef
	onTick     map[Prio]HookDef

	onNewClient map[Prio]HookDef
}

func NewHookConfig() *HookConfig {
	return &HookConfig{
		onConfigLoaded:    make(map[Prio]HookDef),
		onNewLogger:       make(map[Prio]HookDef),
		onNewPool:         make(map[Prio]HookDef),
		onNewProxy:        make(map[Prio]HookDef),
		onNewServer:       make(map[Prio]HookDef),
		onSignal:          make(map[Prio]HookDef),
		onRun:             make(map[Prio]HookDef),
		onBooting:         make(map[Prio]HookDef),
		onBooted:          make(map[Prio]HookDef),
		onOpening:         make(map[Prio]HookDef),
		onOpened:          make(map[Prio]HookDef),
		onClosing:         make(map[Prio]HookDef),
		onClosed:          make(map[Prio]HookDef),
		onTraffic:         make(map[Prio]HookDef),
		onIncomingTraffic: make(map[Prio]Traffic),
		onOutgoingTraffic: make(map[Prio]Traffic),
		onShutdown:        make(map[Prio]HookDef),
		onTick:            make(map[Prio]HookDef),
		onNewClient:       make(map[Prio]HookDef),
	}
}

//nolint:funlen
func (h *HookConfig) AddHook(hookType HookType, prio Prio, hook interface{}) {
	switch hookType {
	case OnConfigLoaded:
		if hookDef, ok := hook.(HookDef); ok {
			h.onConfigLoaded[prio] = hookDef
		}
	case OnNewLogger:
		if hookDef, ok := hook.(HookDef); ok {
			h.onNewLogger[prio] = hookDef
		}
	case OnNewPool:
		if hookDef, ok := hook.(HookDef); ok {
			h.onNewPool[prio] = hookDef
		}
	case OnNewProxy:
		if hookDef, ok := hook.(HookDef); ok {
			h.onNewProxy[prio] = hookDef
		}
	case OnNewServer:
		if hookDef, ok := hook.(HookDef); ok {
			h.onNewServer[prio] = hookDef
		}
	case OnSignal:
		if hookDef, ok := hook.(HookDef); ok {
			h.onSignal[prio] = hookDef
		}
	case OnRun:
		if hookDef, ok := hook.(HookDef); ok {
			h.onRun[prio] = hookDef
		}
	case OnBooting:
		if hookDef, ok := hook.(HookDef); ok {
			h.onBooting[prio] = hookDef
		}
	case OnBooted:
		if hookDef, ok := hook.(HookDef); ok {
			h.onBooted[prio] = hookDef
		}
	case OnOpening:
		if hookDef, ok := hook.(HookDef); ok {
			h.onOpening[prio] = hookDef
		}
	case OnOpened:
		if hookDef, ok := hook.(HookDef); ok {
			h.onOpened[prio] = hookDef
		}
	case OnClosing:
		if hookDef, ok := hook.(HookDef); ok {
			h.onClosing[prio] = hookDef
		}
	case OnClosed:
		if hookDef, ok := hook.(HookDef); ok {
			h.onClosed[prio] = hookDef
		}
	case OnTraffic:
		if hookDef, ok := hook.(HookDef); ok {
			h.onTraffic[prio] = hookDef
		}
	case OnIncomingTraffic:
		if traffic, ok := hook.(Traffic); ok {
			h.onIncomingTraffic[prio] = traffic
		}
	case OnOutgoingTraffic:
		if traffic, ok := hook.(Traffic); ok {
			h.onOutgoingTraffic[prio] = traffic
		}
	case OnShutdown:
		if hookDef, ok := hook.(HookDef); ok {
			h.onShutdown[prio] = hookDef
		}
	case OnTick:
		if hookDef, ok := hook.(HookDef); ok {
			h.onTick[prio] = hookDef
		}
	case OnNewClient:
		if hookDef, ok := hook.(HookDef); ok {
			h.onNewClient[prio] = hookDef
		}
	}
}

//nolint:funlen
func (h *HookConfig) RunHooks(hookType HookType, params ...interface{}) {
	switch hookType {
	case OnConfigLoaded:
		for _, hookDef := range h.onConfigLoaded {
			hookDef(params[0].(*koanf.Koanf))
		}
	case OnNewLogger:
		for _, hookDef := range h.onNewLogger {
			hookDef(params[0].(zerolog.Logger))
		}
	case OnNewPool:
		for _, hookDef := range h.onNewPool {
			hookDef(params[0].(*Pool))
		}
	case OnNewProxy:
		for _, hookDef := range h.onNewProxy {
			hookDef(params[0].(*Proxy))
		}
	case OnNewServer:
		for _, hookDef := range h.onNewServer {
			hookDef(params[0].(*Server))
		}
	case OnSignal:
		for _, hookDef := range h.onSignal {
			hookDef(params[0].(os.Signal))
		}
	case OnRun:
		for _, hookDef := range h.onRun {
			hookDef(params[0].(*Server))
		}
	case OnBooting:
		for _, hookDef := range h.onBooting {
			hookDef(params[0].(*Server), params[1].(gnet.Engine))
		}
	case OnBooted:
		for _, hookDef := range h.onBooted {
			hookDef(params[0].(*Server), params[1].(gnet.Engine))
		}
	case OnOpening:
		for _, hookDef := range h.onOpening {
			hookDef(params[0].(*Server), params[1].(gnet.Conn))
		}
	case OnOpened:
		for _, hookDef := range h.onOpened {
			hookDef(params[0].(*Server), params[1].(gnet.Conn))
		}
	case OnClosing:
		for _, hookDef := range h.onClosing {
			hookDef(params[0].(*Server), params[1].(gnet.Conn), params[2].(error))
		}
	case OnClosed:
		for _, hookDef := range h.onClosed {
			hookDef(params[0].(*Server), params[1].(gnet.Conn), params[2].(error))
		}
	case OnTraffic:
		for _, hookDef := range h.onTraffic {
			hookDef(params[0].(*Server), params[1].(gnet.Conn))
		}
	case OnShutdown:
		for _, hookDef := range h.onShutdown {
			hookDef(params[0].(*Server), params[1].(gnet.Engine))
		}
	case OnTick:
		for _, hookDef := range h.onTick {
			hookDef(params[0].(*Server), params[1].(gnet.Engine))
		}
	case OnNewClient:
		for _, hookDef := range h.onNewClient {
			hookDef(params[0].(*Client), params[1].(*Server))
		}
	}
}

func (h *HookConfig) OnNewClient() map[Prio]HookDef {
	return h.onNewClient
}
