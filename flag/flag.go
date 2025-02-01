package part

import (
	"flag"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type flagType interface {
	string | bool | float64 | int | int32 | int64 | uint | uint32 | uint64 | time.Duration
}

func Lookup[T flagType](name string, defaultVal T) T {
	if f := flag.Lookup(name); f != nil {
		switch any(defaultVal).(type) {
		case string:
			return any(f.Value.String()).(T)
		case bool:
			return any(strings.ToLower(f.Value.String()) == "true").(T)
		case float64:
			if r, e := strconv.ParseFloat(f.Value.String(), 64); e == nil {
				return any(r).(T)
			}
		case int:
			switch unsafe.Sizeof(int(0)) {
			case unsafe.Sizeof(int64(0)):
				if r, e := strconv.ParseInt(f.Value.String(), 0, 64); e == nil {
					return any(int(r)).(T)
				}
			case unsafe.Sizeof(int32(0)):
				if r, e := strconv.ParseInt(f.Value.String(), 0, 32); e == nil {
					return any(int(r)).(T)
				}
			}
		case int32:
			if r, e := strconv.ParseInt(f.Value.String(), 0, 32); e == nil {
				return any(int32(r)).(T)
			}
		case int64:
			if r, e := strconv.ParseInt(f.Value.String(), 0, 64); e == nil {
				return any(r).(T)
			}
		case uint:
			switch unsafe.Sizeof(uint(0)) {
			case unsafe.Sizeof(uint64(0)):
				if r, e := strconv.ParseUint(f.Value.String(), 0, 64); e == nil {
					return any(uint(r)).(T)
				}
			case unsafe.Sizeof(uint32(0)):
				if r, e := strconv.ParseUint(f.Value.String(), 0, 32); e == nil {
					return any(uint(r)).(T)
				}
			}
		case uint32:
			if r, e := strconv.ParseUint(f.Value.String(), 0, 32); e == nil {
				return any(uint32(r)).(T)
			}
		case uint64:
			if r, e := strconv.ParseUint(f.Value.String(), 0, 64); e == nil {
				return any(r).(T)
			}
		case time.Duration:
			if r, e := time.ParseDuration(f.Value.String()); e == nil {
				return any(r).(T)
			}
		}
	}
	return defaultVal
}
