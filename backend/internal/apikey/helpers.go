package apikey

// strPtr — string → *string (nullable column).
func strPtr(s string) *string { return &s }

// derefStr — *string → string (nil → "").
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
