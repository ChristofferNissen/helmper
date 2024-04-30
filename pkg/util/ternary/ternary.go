package ternary

// ternary function
func Ternary[T any](cond bool, val1 T, val2 T) T {
	if cond {
		return val1
	}
	return val2
}
