package act

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/golang-queue/queue/core"
	"github.com/rs/zerolog"
)

type IWorker interface {
	RunFunc() func(ctx context.Context, m core.QueuedMessage) error
}

// Worker keeps track of all policies and actions.
type Worker struct {
	Logger               zerolog.Logger
	DefaultActionTimeout time.Duration

	Actions map[string]*sdkAct.Action
}

var _ IWorker = (*Worker)(nil)

// NewActWorker creates a new act worker.
func NewActWorker(
	worker Worker,
) *Worker {
	if worker.Actions == nil {
		worker.Logger.Warn().Msg("Builtin actions are nil")
		return nil
	}

	for _, action := range worker.Actions {
		if action == nil {
			worker.Logger.Warn().Msg("Action is nil, not adding")
			return nil
		}
		worker.Logger.Debug().Str("name", action.Name).Msg("Registered builtin action")
	}

	return &Worker{
		Logger:               worker.Logger,
		Actions:              worker.Actions,
		DefaultActionTimeout: worker.DefaultActionTimeout,
	}
}

func (w *Worker) RunFunc() func(ctx context.Context, m core.QueuedMessage) error {
	return func(ctx context.Context, m core.QueuedMessage) error {
		actionMessage, ok := m.(*asyncActionMessage)
		if !ok {
			if err := json.Unmarshal(m.Bytes(), &actionMessage); err != nil {
				return fmt.Errorf("failed to unmarshal action message message: %w", err)
			}
		}
		res, err := w.runAction(ctx, actionMessage.Output, actionMessage.Params...)
		if err == nil {
			w.Logger.Debug().Interface("result", res).Msg("Action ran successfully")
		}
		return err
	}
}

// Run runs the function associated with the output.MatchedPolicy and
// returns its result.
func (w *Worker) runAction(
	ctx context.Context,
	output *sdkAct.Output, params ...sdkAct.Parameter,
) (any, *gerr.GatewayDError) {
	// In certain cases, the output may be nil, for example, if the policy
	// evaluation fails. In this case, the run is aborted.
	if output == nil {
		// This should never happen, since the output is always set by the registry
		// to be the default policy if no signals are provided.
		w.Logger.Debug().Msg("Output is nil, run aborted")
		return nil, gerr.ErrNilPointer
	}

	action, ok := w.Actions[output.MatchedPolicy]
	if !ok {
		w.Logger.Warn().Str("matchedPolicy", output.MatchedPolicy).Msg(
			"Action does not exist, run aborted")
		return nil, gerr.ErrActionNotExist
	}

	// Prepend the logger to the parameters.
	params = append([]sdkAct.Parameter{WithLogger(w.Logger)}, params...)

	timeout := w.DefaultActionTimeout
	if action.Timeout > 0 {
		timeout = time.Duration(action.Timeout) * time.Second
	}

	// If the action is asynchronous, run it and return the result immediately.
	if action.Sync {
		w.Logger.Warn().Str("action", action.Name).Msg("Action is synchronous, run aborted")
		return nil, gerr.ErrSyncActionInQueue
	}

	var ctxWithTimeout context.Context
	var cancel context.CancelFunc
	// if timeout is zero, then the context should not have timeout
	if timeout > 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctxWithTimeout, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	return runActionWithTimeout(ctxWithTimeout, action, output, params, w.Logger)
}
