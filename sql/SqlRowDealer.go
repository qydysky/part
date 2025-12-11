package part

import (
	"database/sql"
	"fmt"
	"iter"
	"reflect"
	"strings"
	"unicode/utf8"
)

// 复用结构，当前值只能在迭代中使用
type Row[T any] struct {
	Index int
	Raw   T
	Err   error
}

func DealRows[T any](rows *sql.Rows) ([]T, error) {
	rowNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var res []T

	for rows.Next() {
		rowP := make([]any, len(rowNames))
		for i := 0; i < len(rowNames); i++ {
			rowP[i] = new(any)
		}

		err = rows.Scan(rowP...)
		if err != nil {
			return nil, err
		}

		var (
			stu      = *new(T)
			refV     = reflect.ValueOf(&stu).Elem()
			refT     = reflect.TypeOf(&stu).Elem()
			FieldMap = make(map[string]*reflect.Value)
		)

		for NumField := refV.NumField() - 1; NumField >= 0; NumField-- {
			field := refV.Field(NumField)
			fieldT := refT.Field(NumField)
			fieldTName := fieldT.Name
			if value, ok := fieldT.Tag.Lookup("sql"); ok {
				fieldTName = value
			}
			if !field.IsValid() {
				continue
			}
			if !field.CanSet() {
				FieldMap[strings.ToUpper(fieldTName)] = nil
				continue
			}
			FieldMap[strings.ToUpper(fieldTName)] = &field
		}

		for i := 0; i < len(rowNames); i++ {
			if field, ok := FieldMap[strings.ToUpper(rowNames[i])]; ok {
				if field == nil {
					return nil, fmt.Errorf("DealRows:%s.%s CanSet:false", refT.Name(), rowNames[i])
				}
				val := reflect.ValueOf(*rowP[i].(*any))
				if reflect.TypeOf(*rowP[i].(*any)).ConvertibleTo(field.Type()) {
					field.Set(val)
				} else {
					return nil, fmt.Errorf("DealRows:KindNotMatch:[sql] %v !> [%s.%s] %v", val.Kind(), refT.Name(), rowNames[i], field.Type())
				}
			}
		}
		res = append(res, stu)
	}

	return res, nil
}

func DealRowsIter[T any](rows *sql.Rows) iter.Seq[*Row[T]] {
	var (
		index = 0
		r     = &Row[T]{}
	)
	return func(yield func(*Row[T]) bool) {
		rowNames, err := rows.Columns()
		if err != nil {
			r.Err = err
			if !yield(r) {
				return
			}
		}

		for rows.Next() {
			rowP := make([]any, len(rowNames))
			for i := 0; i < len(rowNames); i++ {
				rowP[i] = new(any)
			}

			err = rows.Scan(rowP...)
			if err != nil {
				r.Err = err
				if !yield(r) {
					break
				}
			}

			var (
				stu      = *new(T)
				refV     = reflect.ValueOf(&stu).Elem()
				refT     = reflect.TypeOf(&stu).Elem()
				FieldMap = make(map[string]*reflect.Value)
			)

			for NumField := refV.NumField() - 1; NumField >= 0; NumField-- {
				field := refV.Field(NumField)
				fieldT := refT.Field(NumField)
				fieldTName := fieldT.Name
				if value, ok := fieldT.Tag.Lookup("sql"); ok {
					fieldTName = value
				}
				if !field.IsValid() {
					continue
				}
				if !field.CanSet() {
					FieldMap[strings.ToUpper(fieldTName)] = nil
					continue
				}
				FieldMap[strings.ToUpper(fieldTName)] = &field
			}

			for i := 0; i < len(rowNames); i++ {
				if field, ok := FieldMap[strings.ToUpper(rowNames[i])]; ok {
					if field == nil {
						r.Err = fmt.Errorf("DealRows:%s.%s CanSet:false", refT.Name(), rowNames[i])
						if !yield(r) {
							return
						}
					}

					val := reflect.ValueOf(*rowP[i].(*any))
					if typ := reflect.TypeOf(*rowP[i].(*any)); typ == nil {
						continue
					} else if typ.ConvertibleTo(field.Type()) {
						field.Set(val)
					} else {
						r.Err = fmt.Errorf("DealRows:KindNotMatch:[sql] %v !> [%s.%s] %v", val.Kind(), refT.Name(), rowNames[i], field.Type())
						if !yield(r) {
							return
						}
					}
				}
			}

			index += 1
			r.Index = index
			r.Err = nil
			r.Raw = stu

			if !yield(r) {
				return
			}
		}
	}
}

func DealRow[T any](rows *sql.Rows) *Row[T] {
	for v := range DealRowsIter[T](rows) {
		return v
	}
	return &Row[T]{
		Err:   nil,
		Raw:   *new(T),
		Index: 0,
	}
}

func DealRowsMapIter(rows *sql.Rows, caseSwitchF ...CaseSwitchF) iter.Seq[*Row[map[string]any]] {
	var (
		index = 0
		r     = &Row[map[string]any]{}
	)
	return func(yield func(*Row[map[string]any]) bool) {
		rowM := make(map[string]any)

		rowNames, err := rows.Columns()
		if err != nil {
			r.Err = err
			if !yield(r) {
				return
			}
		}

		rowP := make([]any, len(rowNames))
		for i := 0; i < len(rowNames); i++ {
			rowP[i] = new(any)
		}

		for rows.Next() {
			clear(rowM)

			err = rows.Scan(rowP...)
			if err != nil {
				r.Err = err
				if !yield(r) {
					return
				}
			}

			for i := 0; i < len(rowNames); i++ {
				if typ := reflect.TypeOf(*rowP[i].(*any)); typ == nil {
					continue
				}
				key := rowNames[i]
				if len(caseSwitchF) > 0 {
					key = caseSwitchF[0](key)
				}
				rowM[key] = reflect.ValueOf(*rowP[i].(*any)).Interface()
			}
			r.Index = index
			r.Err = nil
			r.Raw = rowM
			index += 1
			if !yield(r) {
				return
			}
		}
	}
}

func DealRowMap(rows *sql.Rows, caseSwitchF ...CaseSwitchF) *Row[map[string]any] {
	for v := range DealRowsMapIter(rows, caseSwitchF...) {
		return v
	}
	return &Row[map[string]any]{
		Err:   nil,
		Raw:   make(map[string]any),
		Index: 0,
	}
}

type CaseSwitchF func(string) string

var ToCamel CaseSwitchF = func(s string) string {
	_count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			_count += 1
		}
	}

	var (
		b    strings.Builder
		has_ bool
	)
	b.Grow(len(s) - _count)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			has_ = true
		} else {
			if c >= utf8.RuneSelf {
				// noASCII
			} else if 'a' <= c && c <= 'z' && has_ {
				// a->A
				c -= 'a' - 'A'
			} else if 'A' <= c && c <= 'Z' && !has_ {
				// A->a
				c += 'a' - 'A'
			}
			b.WriteByte(c)
			has_ = false
		}
	}
	return b.String()
}
