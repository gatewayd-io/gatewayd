package metrics

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron"
	promClient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
)

type IMerger interface {
	Add(pluginName string, unixDomainSocket string)
	Remove(pluginName string)
	ReadMetrics() (map[string][]byte, *gerr.GatewayDError)
	MergeMetrics(pluginMetrics map[string][]byte) *gerr.GatewayDError
	Start()
	Stop()
}

type Merger struct {
	scheduler *gocron.Scheduler
	ctx       context.Context //nolint:containedctx

	Logger              zerolog.Logger
	MetricsMergerPeriod time.Duration
	Addresses           map[string]string
	OutputMetrics       []byte
}

var _ IMerger = (*Merger)(nil)

// NewMerger creates a new metrics merger.
func NewMerger(
	ctx context.Context, merger Merger,
) *Merger {
	mergerCtx, span := otel.Tracer(config.TracerName).Start(ctx, "NewMerger")
	defer span.End()

	return &Merger{
		scheduler:           gocron.NewScheduler(time.UTC),
		ctx:                 mergerCtx,
		Logger:              merger.Logger,
		Addresses:           map[string]string{},
		OutputMetrics:       []byte{},
		MetricsMergerPeriod: merger.MetricsMergerPeriod,
	}
}

// Add adds a plugin and its unix domain socket to the map of plugins to merge metrics from.
func (m *Merger) Add(pluginName string, unixDomainSocket string) {
	_, span := otel.Tracer(config.TracerName).Start(m.ctx, "Add")
	defer span.End()

	if _, ok := m.Addresses[pluginName]; ok {
		m.Logger.Warn().Fields(
			map[string]interface{}{
				"plugin": pluginName,
				"socket": unixDomainSocket,
			}).Msg("Plugin already registered, skipping")
		return
	}
	m.Addresses[pluginName] = unixDomainSocket
}

// Remove removes a plugin and its unix domain socket from the map of plugins,
// so that merging metrics don't pick it up on the next scheduled run.
func (m *Merger) Remove(pluginName string) {
	_, span := otel.Tracer(config.TracerName).Start(m.ctx, "Remove")
	defer span.End()

	if _, ok := m.Addresses[pluginName]; !ok {
		m.Logger.Warn().Fields(
			map[string]interface{}{
				"plugin": pluginName,
			}).Msg("Plugin not registered, skipping")
		return
	}
	delete(m.Addresses, pluginName)
}

// ReadMetrics reads metrics from plugins by reading from their unix domain sockets.
//
//nolint:wrapcheck
func (m *Merger) ReadMetrics() (map[string][]byte, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(m.ctx, "ReadMetrics")
	defer span.End()

	pluginMetrics := make(map[string][]byte)

	for pluginName, unixDomainSocket := range m.Addresses {
		if file, err := os.Stat(unixDomainSocket); err != nil || file.IsDir() || file.Mode().Type() != os.ModeSocket {
			continue
		}

		NewHTTPClientOverUDS := func(unixDomainSocket string) http.Client {
			return http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
						var d net.Dialer
						return d.DialContext(ctx, "unix", unixDomainSocket)
					},
				},
			}
		}

		client := NewHTTPClientOverUDS(unixDomainSocket)
		request, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			config.DefaultPluginAddress,
			nil)
		if err != nil {
			span.RecordError(err)
			return nil, gerr.ErrFailedToMergePluginMetrics.Wrap(err)
		}

		response, err := client.Do(request)
		if err != nil {
			span.RecordError(err)
			return nil, gerr.ErrFailedToMergePluginMetrics.Wrap(err)
		}
		defer response.Body.Close()

		metrics, err := io.ReadAll(response.Body)
		if err != nil {
			span.RecordError(err)
			return nil, gerr.ErrFailedToMergePluginMetrics.Wrap(err)
		}

		pluginMetrics[pluginName] = metrics

		span.AddEvent("Read metrics from plugin", trace.WithAttributes(
			attribute.String("plugin", pluginName),
		))
	}

	return pluginMetrics, nil
}

func (m *Merger) MergeMetrics(pluginMetrics map[string][]byte) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(m.ctx, "MergeMetrics")
	defer span.End()

	// TODO: There should be a better, more efficient way to merge metrics from plugins.
	var metricsOutput bytes.Buffer
	enc := expfmt.NewEncoder(io.Writer(&metricsOutput),
		expfmt.NewFormat(expfmt.FormatType(expfmt.TypeTextPlain)))
	for pluginName, metrics := range pluginMetrics {
		// Skip empty metrics.
		if metrics == nil {
			m.Logger.Trace().Str("plugin", pluginName).Msg("Plugin metrics are empty")
			continue
		}

		// Retrieve plugin metrics.
		textParser := expfmt.TextParser{}
		reader := bytes.NewReader(metrics)
		metrics, err := textParser.TextToMetricFamilies(reader)
		if err != nil {
			m.Logger.Trace().Err(err).Msg("Failed to parse plugin metrics")
			span.RecordError(err)
			continue
		}

		// Add plugin label to each metric.
		metricFamilies := map[string]*promClient.MetricFamily{}
		for _, metric := range metrics {
			for _, sample := range metric.GetMetric() {
				// Add plugin label to each metric.
				sample.Label = append(sample.GetLabel(), &promClient.LabelPair{
					Name:  proto.String("plugin"),
					Value: proto.String(strings.ReplaceAll(pluginName, "-", "_")),
				})
			}
			metricFamilies[metric.GetName()] = metric
		}

		// Sort metrics by name and encode them.
		metricNames := maps.Keys(metricFamilies)
		sort.Strings(metricNames)
		for _, metric := range metricNames {
			err := enc.Encode(metricFamilies[metric])
			if err != nil {
				m.Logger.Trace().Err(err).Msg("Failed to encode plugin metrics")
				span.RecordError(err)
				return gerr.ErrFailedToMergePluginMetrics.Wrap(err)
			}
		}

		m.Logger.Trace().Fields(
			map[string]interface{}{
				"plugin": pluginName,
				"count":  len(metricNames),
			}).Msgf("Processed and merged metrics")

		span.AddEvent("Merged metrics from plugin", trace.WithAttributes(
			attribute.String("plugin", pluginName),
		))
	}

	// Update the output metrics.
	m.OutputMetrics = metricsOutput.Bytes()
	return nil
}

// Start starts the metrics merger.
func (m *Merger) Start() {
	ctx, span := otel.Tracer(config.TracerName).Start(m.ctx, "Metrics merger")
	defer span.End()

	startDelay := time.Now().Add(m.MetricsMergerPeriod)
	// Merge metrics from plugins by reading from their unix domain sockets periodically.
	if _, err := m.scheduler.
		Every(m.MetricsMergerPeriod).
		SingletonMode().
		StartAt(startDelay).
		Do(func() {
			_, span := otel.Tracer(config.TracerName).Start(ctx, "Merge metrics")
			span.SetAttributes(attribute.StringSlice("plugins", maps.Keys(m.Addresses)))
			defer span.End()

			m.Logger.Trace().Msg(
				"Running the scheduler for merging metrics from plugins with GatewayD")
			pluginMetrics, err := m.ReadMetrics() //nolint:contextcheck
			if err != nil {
				m.Logger.Error().Err(err.Unwrap()).Msg("Failed to read plugin metrics")
				span.RecordError(err)
				return
			}

			err = m.MergeMetrics(pluginMetrics)
			if err != nil {
				m.Logger.Error().Err(err.Unwrap()).Msg("Failed to merge plugin metrics")
				span.RecordError(err)
			}
		}); err != nil {
		m.Logger.Error().Err(err).Msg("Failed to start metrics merger scheduler")
		span.RecordError(err)
		sentry.CaptureException(err)
	}

	if len(m.Addresses) > 0 {
		m.scheduler.StartAsync()
		m.Logger.Info().Fields(
			map[string]interface{}{
				"startDelay":          startDelay.Format(time.RFC3339),
				"metricsMergerPeriod": m.MetricsMergerPeriod.String(),
			},
		).Msg("Started the metrics merger scheduler")
	}
}

// Stop stops the metrics merger.
func (m *Merger) Stop() {
	_, span := otel.Tracer(config.TracerName).Start(m.ctx, "Stop metrics merger")
	defer span.End()

	m.scheduler.Clear()
}
