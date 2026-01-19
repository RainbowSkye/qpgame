package utils

func Contains[T int | string](arr []T, target T) bool {
	for _, str := range arr {
		if str == target {
			return true
		}
	}
	return false
}
