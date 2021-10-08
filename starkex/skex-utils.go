package starkex

func PiAsString(n int) string {
	if n > len(Pi1024) {
		panic("PiAsString n > 1024")
	}
	return Pi1024[:n]
}
