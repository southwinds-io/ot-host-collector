/*
  host telemetry collector - © 2018-Present - SouthWinds Tech Ltd - www.southwinds.io
  Licensed under the Apache License, Version 2.0 at http://www.apache.org/licenses/LICENSE-2.0
  Contributors to this project, hereby assign copyright in this code to the project,
  to be licensed under the same terms as the rest of the code.
*/

package collector

import (
	"os"

	filexp "southwinds.dev/file-exporter"
	redfish "southwinds.dev/ot_redfish_receiver"

	"github.com/observiq/observiq-otel-collector/processor/resourceattributetransposerprocessor"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	//syslogreceiver is required else filelogreceiver configuration fails saying invalid syslog_parser parser

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/syslogreceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/service"
	"go.uber.org/zap"
)

// NewSettings returns new settings for the collector with default values.
func NewSettings(configPaths []string, version string, loggingOpts []zap.Option) (*service.CollectorSettings, error) {
	// configure receivers
	receiverMap, err := component.MakeReceiverFactoryMap(
		hostmetricsreceiver.NewFactory(),
		redfish.NewFactory(),
		//NOTE:- some how syslog receiver is required to use file log receiver because else
		// file log receiver giving error saying invalid syslog_parser parser configured in telem.yaml
		syslogreceiver.NewFactory(),
		filelogreceiver.NewFactory(),
	)
	if err != nil {
		return nil, err
	}
	// configure processors
	processorMap, err := component.MakeProcessorFactoryMap(
		// add labels to metrics
		resourceattributetransposerprocessor.NewFactory(),
		// add a unique (host.name) to the metric resource(s), allowing users to filter between multiple systems
		resourcedetectionprocessor.NewFactory(),
		// aggregates incoming metrics into a batch, releasing them if a certain time has passed or if a certain number
		// of entries have been aggregated
		batchprocessor.NewFactory(),
	)
	if err != nil {
		return nil, err
	}
	// configure exporters
	exporterMap, err := component.MakeExporterFactoryMap(
		//fe.NewFactory(),
		filexp.NewFactory(),
	)

	if err != nil {
		return nil, err
	}
	buildInfo := component.BuildInfo{
		Command:     os.Args[0],
		Description: "piloth open-telemetry collector for host metrics",
		Version:     version,
	}
	// reads the configuration from a file
	fileP := fileprovider.New()

	configProviderSettings := service.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:       configPaths,
			Providers:  map[string]confmap.Provider{fileP.Scheme(): fileP},
			Converters: []confmap.Converter{expandconverter.New()},
		},
	}
	provider, err := service.NewConfigProvider(configProviderSettings)
	if err != nil {
		return nil, err
	}
	return &service.CollectorSettings{
		Factories: component.Factories{
			Receivers:  receiverMap,
			Processors: processorMap,
			Exporters:  exporterMap,
		},
		BuildInfo:               buildInfo,
		LoggingOptions:          loggingOpts,
		ConfigProvider:          provider,
		DisableGracefulShutdown: true,
	}, nil
}
