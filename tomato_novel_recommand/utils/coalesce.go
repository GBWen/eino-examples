package utils

// Coalesce returns the first non-empty string.
func Coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

