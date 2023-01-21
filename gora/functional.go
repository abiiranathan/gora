package gora

// Loop through any slice or array.
func ForEach[T any](arr []T, f func(val T)) {
	for _, el := range arr {
		f(el)
	}
}

// Same transorm slice. First generic param is array type.
// Second param is the type of the return slice element type.
func MapSlice[T any, V any](arr []T, f func(val T) V) []V {
	result := make([]V, len(arr))
	for index, el := range arr {
		result[index] = f(el)
	}
	return result
}
