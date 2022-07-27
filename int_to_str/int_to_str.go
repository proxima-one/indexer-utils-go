package int_to_str

import (
	"strconv"
	"strings"
)

func IntToStrWithLeadingZeros(n int64, totalLen int) string {
	return StrWithLeadingZeros(strconv.FormatInt(n, 10), totalLen)
}

func StrWithLeadingZeros(s string, totalLen int) string {
	if len(s) < totalLen {
		s = strings.Repeat("0", totalLen-len(s)) + s
	}
	return s
}
