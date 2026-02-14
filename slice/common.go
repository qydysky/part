package part

import (
	"iter"
)

// callback return false to break
func Split[S ~[]T, T comparable](s S, sep S, callback func(s S) bool) {
	var p int
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(sep); j++ {
			if s[i] == sep[j] {
				if !callback(s[p:i]) {
					return
				}
				p = i + 1
				break
			}
		}
	}
	if !callback(s[p:]) {
		return
	}
}

// callback return false to break
func SplitAfter[S ~[]T, T comparable](s S, sep S, callback func(s S) bool) {
	var p int
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(sep); j++ {
			if s[i] == sep[j] {
				if !callback(s[p:i]) {
					return
				}
				p = i
				break
			}
		}
	}
	if !callback(s[p:]) {
		return
	}
}

func DelFront[S ~[]T, T any](s *S, beforeIndex int) {
	*s = (*s)[:copy(*s, (*s)[beforeIndex:])]
}

func AddFront[S ~[]T, T any](s *S, t *T) {
	*s = append(*s, *new(T))
	*s = (*s)[:1+copy((*s)[1:], *s)]
	(*s)[0] = *t
}

func DelBack[S ~[]T, T any](s *S, fromIndex int) {
	*s = (*s)[:fromIndex]
}

func AddBack[S ~[]T, T any](s *S, t *T) {
	*s = append(*s, *t)
}

func LoopAddBack[S ~[]T, T any](s *S, t *T) {
	DelFront(s, 1)
	AddBack(s, t)
}

func LoopAddFront[S ~[]T, T any](s *S, t *T) {
	DelBack(s, len(*s)-1)
	AddFront(s, t)
}

func Resize[S ~[]T, T any](s *S, size int) {
	if len(*s) >= size || cap(*s) >= size {
		*s = (*s)[:size]
	} else {
		*s = append((*s)[:cap(*s)], make([]T, size-cap(*s))...)
	}
}

func Del[S ~[]T, T any](s *S, f func(t *T) (del bool)) {
	for i := 0; i < len(*s); i++ {
		if f(&(*s)[i]) {
			*s = append((*s)[:i], (*s)[i+1:]...)
			i -= 1
		}
	}
}

func DelPtr[S ~[]*T, T any](s *S, f func(t *T) (del bool)) {
	for i := 0; i < len(*s); i++ {
		if f((*s)[i]) {
			*s = append((*s)[:i], (*s)[i+1:]...)
			i -= 1
		}
	}
}

func Range[T any](s []T) iter.Seq2[int, *T] {
	return func(yield func(int, *T) bool) {
		for i := 0; i < len(s); i++ {
			if !yield(i, &(s)[i]) {
				return
			}
		}
	}
}

func Search[T any](s []T, okf func(*T) bool) (k int, t *T) {
	for i := 0; i < len(s); i++ {
		if okf(&(s)[i]) {
			return i, &(s)[i]
		}
	}
	return -1, nil
}

// T是ptr时，使用AppendPtr
func Append[T any](s *[]T) *T {
	c, l := cap(*s), len(*s)
	if c > l {
		*s = (*s)[:l+1]
	} else {
		*s = append(*s, *new(T))
	}
	return &(*s)[l]
}

func AppendPtr[T any](s *[]*T) *T {
	c, l := cap(*s), len(*s)
	if c > l {
		*s = (*s)[:l+1]
		if (*s)[l] == nil {
			(*s)[l] = new(T)
		}
	} else {
		*s = append(*s, new(T))
	}
	return (*s)[l]
}
