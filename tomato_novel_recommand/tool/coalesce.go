package tools

// coalesce 返回第一个非空字符串。
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
