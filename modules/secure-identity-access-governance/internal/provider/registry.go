package provider

import (
	"fmt"
	"sort"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
)

type Registry struct {
	descriptors []model.ProviderDescriptor
	configs     map[string]config.ProviderConfig
}

func New(configs []config.ProviderConfig) (*Registry, error) {
	descriptors := make([]model.ProviderDescriptor, 0, len(configs))
	seen := make(map[string]struct{}, len(configs))
	for _, item := range configs {
		if _, exists := seen[item.Name]; exists {
			return nil, fmt.Errorf("duplicate provider %q", item.Name)
		}
		seen[item.Name] = struct{}{}
		status := "disabled"
		if item.Enabled {
			status = "declared"
		}
		capabilities := append([]string(nil), item.Capabilities...)
		sort.Strings(capabilities)
		descriptors = append(descriptors, model.ProviderDescriptor{
			Name:         item.Name,
			Kind:         item.Kind,
			Status:       status,
			Capabilities: capabilities,
			Exportable:   item.Exportable,
			Interactive:  item.Interactive,
		})
	}
	sort.Slice(descriptors, func(i, j int) bool { return descriptors[i].Name < descriptors[j].Name })
	configsByName := make(map[string]config.ProviderConfig, len(configs))
	for _, item := range configs {
		item.Capabilities = append([]string(nil), item.Capabilities...)
		sort.Strings(item.Capabilities)
		configsByName[item.Name] = item
	}
	return &Registry{descriptors: descriptors, configs: configsByName}, nil
}

func (r *Registry) Configuration(name string) (config.ProviderConfig, bool) {
	item, ok := r.configs[name]
	item.Capabilities = append([]string(nil), item.Capabilities...)
	return item, ok
}

func (r *Registry) Descriptors() []model.ProviderDescriptor {
	result := make([]model.ProviderDescriptor, len(r.descriptors))
	copy(result, r.descriptors)
	for i := range result {
		result[i].Capabilities = append([]string(nil), result[i].Capabilities...)
	}
	return result
}
