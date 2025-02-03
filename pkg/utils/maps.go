package utils

import "sort"

// GetFirstItemFromMap returns the first item from a map, in a deterministic order
func GetFirstItemFromMap(m map[string]string) (string, string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		return "", ""
	}
	return keys[0], m[keys[0]]
}
