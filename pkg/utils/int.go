package utils

import (
	"math/big"
	"strconv"
	"strings"
)

func IntToStrWithLeadingZeros(n int64, totalLen int) string {
	return PrependZeros(strconv.FormatInt(n, 10), totalLen)
}

func PrependZeros(s string, requiredLen int) string {
	if len(s) < requiredLen {
		s = strings.Repeat("0", requiredLen-len(s)) + s
	}
	return s
}

func StringToBigInt(str string) *big.Int {
	var res big.Int
	res.SetString(str, 10)
	return &res
}

func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func MustConvStrToInt64(s string) int64 {
	res, err := strconv.ParseInt(s, 10, 64)
	PanicOnError(err)
	return res
}
