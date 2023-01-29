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

	"github.com/go-co-op/gocron"
	promClient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
)

type IMerger interface {
	Add(pluginName string, unixDomainSocket string)
	ReadMetrics() (map[string]io.ReadCloser, error)
	Start()
	Stop()
}

type Merger struct {
	metricsMergerScheduler *gocron.Scheduler

	Logger              zerolog.Logger
	MetricsMergerPeriod time.Duration
	Addresses           map[string]string
	OutputMetrics       []byte
}

var _ IMerger = &Merger{}

func NewMerger(metricsMergerPeriod time.Duration, logger zerolog.Logger) *Merger {
	return &Merger{
		metricsMergerScheduler: gocron.NewScheduler(time.UTC),
		Logger:                 logger,
		Addresses:              map[string]string{},
		OutputMetrics:          []byte{},
		MetricsMergerPeriod:    metricsMergerPeriod,
	}
}

func (m *Merger) Add(pluginName string, unixDomainSocket string) {
	if _, ok := m.Addresses[pluginName]; ok {
		m.Logger.Warn().Fields(
			map[string]interface{}{
				"plugin": pluginName,
				"socket": unixDomainSocket,
			}).Msg("Plugin already registered")
		return
	}
	m.Addresses[pluginName] = unixDomainSocket
}

func (m *Merger) ReadMetrics() (map[string]io.ReadCloser, error) {
	readers := make(map[string]io.ReadCloser)

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
		response, err := client.Get("http://plugins/metrics")
		if err != nil {
			return nil, err
		}
		readers[pluginName] = response.Body
	}

	return readers, nil
}

func (m *Merger) Start() {
	// Merge metrics from plugins by reading from their unix domain sockets.
	// This is done periodically.

	m.metricsMergerScheduler.Every(m.MetricsMergerPeriod).SingletonMode().StartAt(time.Now().Add(m.MetricsMergerPeriod)).Do(func() {
		readers, err := m.ReadMetrics()
		if err != nil {
			m.Logger.Error().Err(err).Msg("Failed to read plugin metrics")
		}

		// TODO: There should be a better, more efficient way to merge metrics from plugins.
		var metricsOutput bytes.Buffer
		enc := expfmt.NewEncoder(io.Writer(&metricsOutput), expfmt.FmtText)
		for plugin, reader := range readers {
			if reader != nil {
				defer reader.Close()

				// Retrieve plugin metrics.
				textParser := expfmt.TextParser{}
				if metrics, err := textParser.TextToMetricFamilies(reader); err != nil {
					m.Logger.Error().Err(err).Msg("Failed to parse plugin metrics")
				} else {
					metricFamilies := map[string]*promClient.MetricFamily{}

					for _, metric := range metrics {
						for _, sample := range metric.Metric {
							// Add plugin label to each metric.
							sample.Label = append(sample.Label, &promClient.LabelPair{
								Name:  proto.String("plugin"),
								Value: proto.String(strings.ReplaceAll(plugin, "-", "_")),
							})
						}
						metricFamilies[metric.GetName()] = metric
					}

					metricNames := maps.Keys(metricFamilies)
					sort.Strings(metricNames)

					for _, metric := range metricNames {
						err := enc.Encode(metricFamilies[metric])
						if err != nil {
							m.Logger.Error().Err(err).Msg("Failed to encode plugin metrics")
							return
						}
					}

					m.Logger.Debug().Fields(
						map[string]interface{}{
							"plugin": plugin,
							"count":  len(metricNames),
						}).Msgf("Processed and merged metrics")
				}
			}
		}

		m.OutputMetrics = metricsOutput.Bytes()
	})

	m.metricsMergerScheduler.StartAsync()
}

func (m *Merger) Stop() {
	m.metricsMergerScheduler.Clear()
}
