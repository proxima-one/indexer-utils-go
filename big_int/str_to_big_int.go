package helpers

import (
	"math/big"
)

func StringToBigInt(str string) *big.Int {
	var res big.Int
	res.SetString(str, 10)
	return &res
}
