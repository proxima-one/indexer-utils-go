package panic_on_error

import "fmt"

func PanicOnError(err error) {
	if err != nil {
		panic(fmt.Sprint("Error:", err))
	}
}
