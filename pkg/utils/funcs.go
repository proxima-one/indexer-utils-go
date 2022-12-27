package utils

import "golang.org/x/exp/constraints"

func Min[T constraints.Ordered](x, y T) T {
	if x < y {
		return x
	}
	return y
}

func Max[T constraints.Ordered](x, y T) T {
	if x > y {
		return x
	}
	return y
}

func MapArray[A any, B any](arr []A, fun func(A) B) (res []B) {
	for _, elem := range arr {
		res = append(res, fun(elem))
	}
	return
}

func FilterArray[A any](arr []A, fun func(A) bool) (res []A) {
	for _, elem := range arr {
		if fun(elem) {
			res = append(res, elem)
		}
	}
	return
}

func MapToSlice[K comparable, V any](m map[K]V) []V {
	res := make([]V, 0, len(m))
	for _, val := range m {
		res = append(res, val)
	}
	return res
}
