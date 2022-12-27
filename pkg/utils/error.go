package utils

import "fmt"

func PanicOnError(err error) {
	if err != nil {
		panic(fmt.Sprint("Error:", err.Error()))
	}
}
