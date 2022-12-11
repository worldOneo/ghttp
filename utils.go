package ghttp

// Parses the bytes to int.
// This function contains no checks and might yield
// garbage when not provided with an int.
func BytesToInt(bytes []byte) int64 {
	var a int64
	for _, b := range bytes {
		a *= 10
		a += int64(b - '0')
	}
	return a
}

// CopyBytes copies the provided bytes into a new
// independent byte slice.
func CopyBytes(bytes []byte) []byte {
	c := make([]byte, len(bytes))
	copy(c, bytes)
	return c
}

// Copy string clones the given string into a new
// independent string.
func CopyString(s string) string {
	return string(CopyBytes([]byte(s)))
}
