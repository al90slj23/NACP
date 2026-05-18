package unitplatform

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	adaptersMu sync.RWMutex
	adapters   = map[string]Adapter{}
	order      []string
)

func Register(adapter Adapter) {
	if adapter == nil {
		return
	}
	platformType := normalizeType(adapter.Type())
	if platformType == "" {
		return
	}
	adaptersMu.Lock()
	defer adaptersMu.Unlock()
	if _, exists := adapters[platformType]; !exists {
		order = append(order, platformType)
	}
	adapters[platformType] = adapter
	for _, alias := range adapter.Aliases() {
		normalized := normalizeType(alias)
		if normalized != "" {
			adapters[normalized] = adapter
		}
	}
}

func Get(platformType string) (Adapter, bool) {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()
	adapter, ok := adapters[normalizeType(platformType)]
	return adapter, ok
}

func MustGet(platformType string) (Adapter, error) {
	adapter, ok := Get(platformType)
	if !ok {
		return nil, fmt.Errorf("当前单位类型 %s 暂不支持", platformType)
	}
	return adapter, nil
}

func Adapters() []Adapter {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()
	platformTypes := append([]string(nil), order...)
	sort.SliceStable(platformTypes, func(i, j int) bool {
		return detectPriority(platformTypes[i]) < detectPriority(platformTypes[j])
	})
	result := make([]Adapter, 0, len(platformTypes))
	for _, platformType := range platformTypes {
		if adapter, ok := adapters[platformType]; ok {
			result = append(result, adapter)
		}
	}
	return result
}

func PlatformTypes() []string {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()
	types := append([]string(nil), order...)
	sort.Strings(types)
	return types
}

func normalizeType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func detectPriority(platformType string) int {
	switch normalizeType(platformType) {
	case "rixapi":
		return 10
	case "shellapi":
		return 20
	case "veloera":
		return 30
	case "onehub":
		return 40
	case "donehub":
		return 50
	case "anyrouter":
		return 60
	case "sub2api":
		return 70
	case "cliproxyapi":
		return 80
	case "openai":
		return 90
	case "claude":
		return 100
	case "gemini":
		return 110
	case "geminicli":
		return 120
	case "antigravity":
		return 130
	case "codex":
		return 140
	case "newapi":
		return 900
	case "oneapi":
		return 910
	case "oneapifork":
		return 990
	default:
		return 500
	}
}
