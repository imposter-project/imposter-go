package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// findByBasePath returns the single config whose BasePath matches, failing the
// test if there is not exactly one. BasePath is used purely as a label here.
func findByBasePath(t *testing.T, configs []Config, basePath string) Config {
	t.Helper()
	var found []Config
	for _, c := range configs {
		if c.BasePath == basePath {
			found = append(found, c)
		}
	}
	require.Len(t, found, 1, "expected exactly one config labelled %q", basePath)
	return found[0]
}

func TestCoalesceWebSocketConfigs_SingleConfigUnchanged(t *testing.T) {
	in := []Config{
		{Plugin: "websocket", BasePath: "ws", Resources: []Resource{{}, {}}},
	}

	out := coalesceWebSocketConfigs(in)

	require.Len(t, out, 1)
	require.Equal(t, "websocket", out[0].Plugin)
	require.Len(t, out[0].Resources, 2, "a lone websocket config must pass through untouched")
}

func TestCoalesceWebSocketConfigs_NoWebSocketConfigs(t *testing.T) {
	in := []Config{
		{Plugin: "rest", BasePath: "a"},
		{Plugin: "soap", BasePath: "b"},
	}

	out := coalesceWebSocketConfigs(in)

	require.Len(t, out, 2, "configs without any websocket plugin are returned as-is")
	require.Equal(t, "a", out[0].BasePath)
	require.Equal(t, "b", out[1].BasePath)
}

func TestCoalesceWebSocketConfigs_MergesResourcesAcrossFiles(t *testing.T) {
	in := []Config{
		{Plugin: "websocket", BasePath: "ws1", Resources: []Resource{{}, {}}},
		{Plugin: "websocket", BasePath: "ws2", Resources: []Resource{{}}},
		{Plugin: "websocket", BasePath: "ws3", Resources: []Resource{{}, {}, {}}},
	}

	out := coalesceWebSocketConfigs(in)

	require.Len(t, out, 1, "all websocket configs collapse into one")
	require.Equal(t, "ws1", out[0].BasePath, "the merged config keeps the first one's position/identity")
	require.Len(t, out[0].Resources, 6, "resources are the union of every websocket file")
}

func TestCoalesceWebSocketConfigs_PreservesOrderAmongOtherPlugins(t *testing.T) {
	in := []Config{
		{Plugin: "rest", BasePath: "restA"},
		{Plugin: "websocket", BasePath: "ws1", Resources: []Resource{{}}},
		{Plugin: "rest", BasePath: "restB"},
		{Plugin: "websocket", BasePath: "ws2", Resources: []Resource{{}, {}}},
	}

	out := coalesceWebSocketConfigs(in)

	// rest configs are untouched and keep their relative order; the merged
	// websocket config stays where the first websocket config was (index 1).
	require.Len(t, out, 3)
	require.Equal(t, "restA", out[0].BasePath)
	require.Equal(t, "websocket", out[1].Plugin)
	require.Equal(t, "ws1", out[1].BasePath)
	require.Len(t, out[1].Resources, 3)
	require.Equal(t, "restB", out[2].BasePath)
}

func TestCoalesceWebSocketConfigs_MergesInterceptorsAndSchedules(t *testing.T) {
	in := []Config{
		{
			Plugin:       "websocket",
			BasePath:     "ws1",
			Interceptors: []Interceptor{{}},
			Schedules:    []Schedule{{}},
		},
		{
			Plugin:       "websocket",
			BasePath:     "ws2",
			Interceptors: []Interceptor{{}, {}},
			Schedules:    []Schedule{{}, {}, {}},
		},
	}

	out := coalesceWebSocketConfigs(in)

	require.Len(t, out, 1)
	require.Len(t, out[0].Interceptors, 3, "interceptors from every websocket file are merged")
	require.Len(t, out[0].Schedules, 4, "schedules from every websocket file are merged")
}

func TestCoalesceWebSocketConfigs_MergesSystemStoresFromEachFile(t *testing.T) {
	in := []Config{
		{
			Plugin:   "websocket",
			BasePath: "ws1",
			System:   &System{Stores: map[string]StoreDefinition{"a": {PreloadFile: "a.json"}}},
		},
		{
			Plugin:   "websocket",
			BasePath: "ws2",
			System:   &System{Stores: map[string]StoreDefinition{"b": {PreloadFile: "b.json"}}},
		},
	}

	out := coalesceWebSocketConfigs(in)

	require.Len(t, out, 1)
	require.NotNil(t, out[0].System)
	require.Len(t, out[0].System.Stores, 2, "each split file may declare its own store")
	require.Equal(t, "a.json", out[0].System.Stores["a"].PreloadFile)
	require.Equal(t, "b.json", out[0].System.Stores["b"].PreloadFile)
}

func TestMergeSystem_ExtraNilLeavesBaseUntouched(t *testing.T) {
	base := &Config{System: &System{Stores: map[string]StoreDefinition{"a": {}}}}

	mergeSystem(base, nil)

	require.Len(t, base.System.Stores, 1, "a nil extra System is a no-op")
}

func TestMergeSystem_AdoptsExtraWhenBaseHasNone(t *testing.T) {
	base := &Config{}
	extra := &System{Stores: map[string]StoreDefinition{"a": {PreloadFile: "a.json"}}}

	mergeSystem(base, extra)

	require.Same(t, extra, base.System, "base with no System adopts extra directly")
}

func TestMergeSystem_MergesIntoNilMaps(t *testing.T) {
	// base.System exists but its maps are nil, exercising the lazy-init branches.
	base := &Config{System: &System{}}
	extra := &System{
		Stores:        map[string]StoreDefinition{"a": {PreloadFile: "a.json"}},
		XMLNamespaces: map[string]string{"ns": "urn:example"},
	}

	mergeSystem(base, extra)

	require.Equal(t, "a.json", base.System.Stores["a"].PreloadFile)
	require.Equal(t, "urn:example", base.System.XMLNamespaces["ns"])
}

func TestMergeSystem_MergesStoresAndNamespaces(t *testing.T) {
	base := &Config{System: &System{
		Stores:        map[string]StoreDefinition{"a": {PreloadFile: "a.json"}},
		XMLNamespaces: map[string]string{"ns1": "urn:one"},
	}}
	extra := &System{
		Stores:        map[string]StoreDefinition{"b": {PreloadFile: "b.json"}},
		XMLNamespaces: map[string]string{"ns2": "urn:two"},
	}

	mergeSystem(base, extra)

	require.Len(t, base.System.Stores, 2)
	require.Equal(t, "a.json", base.System.Stores["a"].PreloadFile)
	require.Equal(t, "b.json", base.System.Stores["b"].PreloadFile)
	require.Equal(t, "urn:one", base.System.XMLNamespaces["ns1"])
	require.Equal(t, "urn:two", base.System.XMLNamespaces["ns2"])
}

func TestMergeSystem_ExtraOverwritesOnStoreNameClash(t *testing.T) {
	base := &Config{System: &System{Stores: map[string]StoreDefinition{"shared": {PreloadFile: "base.json"}}}}
	extra := &System{Stores: map[string]StoreDefinition{"shared": {PreloadFile: "extra.json"}}}

	mergeSystem(base, extra)

	require.Len(t, base.System.Stores, 1)
	require.Equal(t, "extra.json", base.System.Stores["shared"].PreloadFile,
		"on a store-name clash the later file wins")
}
