package utils

import "strings"

// StringSliceContainsElement checks if a string slice contains a specific element
func StringSliceContainsElement(slice *[]string, element string) bool {
	for _, s := range *slice {
		if s == element {
			return true
		}
	}
	return false
}

// RemoveEmptyStrings removes empty or space-only strings from a string slice
func RemoveEmptyStrings(slice []string) []string {
	var result []string
	for _, s := range slice {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
