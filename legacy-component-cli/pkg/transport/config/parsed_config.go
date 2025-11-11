package config

import (
	"encoding/json"
	"fmt"
	"os"

	"sigs.k8s.io/yaml"

	cdv2 "github.com/gardener/landscaper/legacy-component-spec/bindings-go/apis/v2"

	"github.com/gardener/landscaper/legacy-component-cli/pkg/transport/filters"
)

type ParsedTransportConfig struct {
	Downloaders     []ParsedDownloaderDefinition
	Processors      []ParsedProcessorDefinition
	Uploaders       []ParsedUploaderDefinition
	ProcessingRules []ParsedProcessingRuleDefinition
}

type ParsedDownloaderDefinition struct {
	Name    string
	Type    string
	Spec    *json.RawMessage
	Filters []filters.Filter
}

type ParsedProcessorDefinition struct {
	Name string
	Type string
	Spec *json.RawMessage
}

type ParsedUploaderDefinition struct {
	Name    string
	Type    string
	Spec    *json.RawMessage
	Filters []filters.Filter
}

type ParsedProcessingRuleDefinition struct {
	Name       string
	Processors []ParsedProcessorDefinition
	Filters    []filters.Filter
}

// ParseTransportConfig loads and parses a transport config file
func ParseTransportConfig(configFilePath string) (*ParsedTransportConfig, error) {
	transportCfgYaml, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read transport config file: %w", err)
	}

	var config transportConfig
	if err := yaml.Unmarshal(transportCfgYaml, &config); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transport config: %w", err)
	}

	var parsedConfig ParsedTransportConfig
	ff := filters.NewFilterFactory()

	// downloaders
	for _, downloaderDefinition := range config.Downloaders {
		filters, err := createFilterList(downloaderDefinition.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("unable to create filters for downloader %s: %w", downloaderDefinition.Name, err)
		}
		parsedConfig.Downloaders = append(parsedConfig.Downloaders, ParsedDownloaderDefinition{
			Name:    downloaderDefinition.Name,
			Type:    downloaderDefinition.Type,
			Spec:    downloaderDefinition.Spec,
			Filters: filters,
		})
	}

	// processors
	for _, processorsDefinition := range config.Processors {
		parsedConfig.Processors = append(parsedConfig.Processors, ParsedProcessorDefinition{
			Name: processorsDefinition.Name,
			Type: processorsDefinition.Type,
			Spec: processorsDefinition.Spec,
		})
	}

	// uploaders
	for _, uploaderDefinition := range config.Uploaders {
		filters, err := createFilterList(uploaderDefinition.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("unable to create filters for uploader %s: %w", uploaderDefinition.Name, err)
		}
		parsedConfig.Uploaders = append(parsedConfig.Uploaders, ParsedUploaderDefinition{
			Name:    uploaderDefinition.Name,
			Type:    uploaderDefinition.Type,
			Spec:    uploaderDefinition.Spec,
			Filters: filters,
		})
	}

	// processing rules
	for _, processingRule := range config.ProcessingRules {
		filters, err := createFilterList(processingRule.Filters, ff)
		if err != nil {
			return nil, fmt.Errorf("unable to create filters for processing rule %s: %w", processingRule.Name, err)
		}

		processors := []ParsedProcessorDefinition{}
		for _, processorName := range processingRule.Processors {
			processorDefined, err := findProcessorByName(processorName.Name, &parsedConfig)
			if err != nil {
				return nil, fmt.Errorf("unable to parse processing rule %s: %w", processingRule.Name, err)
			}
			processors = append(processors, *processorDefined)
		}

		parsedProcessingRule := ParsedProcessingRuleDefinition{
			Name:       processingRule.Name,
			Processors: processors,
			Filters:    filters,
		}

		parsedConfig.ProcessingRules = append(parsedConfig.ProcessingRules, parsedProcessingRule)
	}

	return &parsedConfig, nil
}

// MatchDownloaders finds all matching downloaders
func (c *ParsedTransportConfig) MatchDownloaders(cd cdv2.ComponentDescriptor, res cdv2.Resource) []ParsedDownloaderDefinition {
	dls := []ParsedDownloaderDefinition{}
	for _, downloader := range c.Downloaders {
		if areAllFiltersMatching(downloader.Filters, cd, res) {
			dls = append(dls, downloader)
		}
	}
	return dls
}

// MatchUploaders finds all matching uploaders
func (c *ParsedTransportConfig) MatchUploaders(cd cdv2.ComponentDescriptor, res cdv2.Resource) []ParsedUploaderDefinition {
	uls := []ParsedUploaderDefinition{}
	for _, uploader := range c.Uploaders {
		if areAllFiltersMatching(uploader.Filters, cd, res) {
			uls = append(uls, uploader)
		}
	}
	return uls
}

// MatchProcessingRules finds all matching processing rules
func (c *ParsedTransportConfig) MatchProcessingRules(cd cdv2.ComponentDescriptor, res cdv2.Resource) []ParsedProcessingRuleDefinition {
	prs := []ParsedProcessingRuleDefinition{}
	for _, processingRule := range c.ProcessingRules {
		if areAllFiltersMatching(processingRule.Filters, cd, res) {
			prs = append(prs, processingRule)
		}
	}
	return prs
}

func areAllFiltersMatching(filters []filters.Filter, cd cdv2.ComponentDescriptor, res cdv2.Resource) bool {
	for _, filter := range filters {
		if !filter.Matches(cd, res) {
			return false
		}
	}
	return true
}

func findProcessorByName(name string, lookup *ParsedTransportConfig) (*ParsedProcessorDefinition, error) {
	for _, processor := range lookup.Processors {
		if processor.Name == name {
			return &processor, nil
		}
	}
	return nil, fmt.Errorf("unable to find processor %s", name)
}

func createFilterList(filterDefinitions []filterDefinition, ff *filters.FilterFactory) ([]filters.Filter, error) {
	var filters []filters.Filter
	for _, f := range filterDefinitions {
		filter, err := ff.Create(f.Type, f.Spec)
		if err != nil {
			return nil, fmt.Errorf("error creating filter list for type %s with args %s: %w", f.Type, string(*f.Spec), err)
		}
		filters = append(filters, filter)
	}
	return filters, nil
}
