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
	res := make([]V, len(m))
	i := 0
	for _, val := range m {
		res[i] = val
		i++
	}
	return res
}

func MapKeys[K comparable, V any](m map[K]V) []K {
	res := make([]K, len(m))
	i := 0
	for key := range m {
		res[i] = key
		i++
	}
	return res
}
