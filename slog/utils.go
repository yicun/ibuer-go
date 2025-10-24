// Package slog provides field-level JSON logging, outputting only fields with the 'slog' tag.
// Priority: Struct Logger → Field Logger → ser=xxx → Basic Type → Mask
package slog

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ----- Tag Parsing -----

type tagOpts string

func (o tagOpts) Get(key string) string {
	for _, seg := range strings.Split(string(o), ",") {
		seg = strings.TrimSpace(seg)
		if kv := strings.SplitN(seg, "=", 2); len(kv) == 2 && strings.TrimSpace(kv[0]) == key {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

func (o tagOpts) Contains(opt string) bool {
	for _, seg := range strings.Split(string(o), ",") {
		if strings.TrimSpace(seg) == opt {
			return true
		}
	}
	return false
}

// ----- Utility Functions -----

// IsZeroer 是一个用户自定义接口，用于判断类型是否为空/零值
type IsZeroer interface {
	IsZero() bool
}

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		// 对于实现 IsZeroer 接口的结构体，直接判断
		if t, ok := v.Interface().(IsZeroer); ok {
			return t.IsZero()
		}
		// 对于结构体，递归检查所有导出字段是否都为空
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			// 只检查导出字段 (字段名首字母大写)
			if field.PkgPath == "" {
				if !isEmpty(v.Field(i)) {
					return false
				}
			}
		}
		return true
	}
	return false
}

func defaultMask(s string) string {
	if len(s) <= 4 {
		return "****"
	}

	// For longer strings, preserve more characters for better readability
	visibleChars := 2
	if len(s) > 8 {
		visibleChars = 3 // Long strings preserve more characters
	}

	prefix := s[:visibleChars/2]
	suffix := s[len(s)-visibleChars/2:]
	maskedLength := len(s) - visibleChars

	if maskedLength <= 0 {
		return "****"
	}

	return prefix + strings.Repeat("*", maskedLength) + suffix
}

func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	}
	return 0, fmt.Errorf("cannot convert %T to float64", v)
}
