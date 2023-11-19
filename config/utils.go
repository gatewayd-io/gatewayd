package config

// If returns truthy if predicate is true, falsy otherwise.
func If[T any](predicate bool, truthy, falsy T) T {
	if predicate {
		return truthy
	}
	return falsy
}

// Exists returns true if key exists in map_, false otherwise.
func Exists[T comparable, V any](map_ map[T]V, key T) bool {
	_, ok := map_[key]
	return ok
}
