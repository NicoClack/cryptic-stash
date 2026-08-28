package tempkeyvalue

import "github.com/NicoClack/cryptic-stash/backend/common"

type RegistryGroup struct {
	Registry *Registry
	Path     string
}

func (registry *Registry) Group(relativePath string) *RegistryGroup {
	return &RegistryGroup{
		Path:     relativePath,
		Registry: registry,
	}
}

func (group *RegistryGroup) Group(relativePath string) *RegistryGroup {
	return &RegistryGroup{
		Path:     common.JoinPaths(group.Path, relativePath),
		Registry: group.Registry,
	}
}

func (group *RegistryGroup) Register(definition *Definition) {
	definition.Name = common.JoinPaths(group.Path, definition.Name)
	group.Registry.Register(definition)
}
