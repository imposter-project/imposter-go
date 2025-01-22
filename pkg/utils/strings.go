package utils

// StringSliceContainsElement checks if a string slice contains a specific element
func StringSliceContainsElement(slice *[]string, element string) bool {
	for _, s := range *slice {
		if s == element {
			return true
		}
	}
	return false
}
