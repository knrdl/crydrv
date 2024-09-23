package main

import (
	"errors"
	"os"

	"golang.org/x/exp/constraints"
)

func Check(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func Try[T any](result T, err error) T {
	Check(err)
	return result
}

func Clamp[T constraints.Ordered](low, value, high T) T {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func Min[T constraints.Ordered](values ...T) T {
	output := values[0]
	for _, value := range values {
		if value < output {
			output = value
		}
	}
	return output
}

func IsFile(path string) (bool, error) {
	if fi, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		} else {
			return false, err
		}
	} else {
		return !fi.IsDir(), nil
	}
}

// func IsDir(path string) (bool, error) {
// 	if fi, err := os.Stat(path); err != nil {
// 		if errors.Is(err, os.ErrNotExist) {
// 			return false, nil
// 		} else {
// 			return false, err
// 		}
// 	} else {
// 		return fi.IsDir(), nil
// 	}
// }

// func All[T any](s []T, fn func(T) bool) bool {
// 	for _, e := range s {
// 		if !fn(e) {
// 			return false
// 		}
// 	}
// 	return true
// }

// func Any[T any](s []T, fn func(T) bool) bool {
// 	for _, e := range s {
// 		if fn(e) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func Filter[T any](s []T, fn func(T) bool) []T {
// 	ret := make([]T, 0)
// 	for _, e := range s {
// 		if fn(e) {
// 			ret = append(ret, e)
// 		}
// 	}
// 	return ret
// }

// func Map[T1, T2 any](s []T1, fn func(T1) T2) []T2 {
// 	ret := make([]T2, 0, len(s))
// 	for _, e := range s {
// 		ret = append(ret, fn(e))
// 	}
// 	return ret
// }
