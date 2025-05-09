package part

func Contains[T comparable, S any](s map[T]S, keys ...T) (missKey []T) {
	for _, tk := range keys {
		if _, ok := s[tk]; !ok {
			missKey = append(missKey, tk)
		}
	}
	return
}
