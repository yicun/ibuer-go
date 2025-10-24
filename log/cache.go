// Package log provides field-level JSON logging, outputting only fields with the 'log' tag.
// Priority: Struct Logger → Field Logger → ser=xxx → Basic Type → Mask
package log

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// ----- Cache Structures -----

type structCache struct {
	sync.RWMutex
	m map[reflect.Type]*structInfo
}

type structInfo struct {
	hasLogTag bool
	fields    []fieldInfo
}

type fieldInfo struct {
	index    int
	name     string
	opts     fieldOptions
	jsonName string
	jsonOpts tagOpts
}

type fieldOptions struct {
	Name       string
	OmitEmpty  bool
	Serializer string // Changed from 'func=' to 'ser='
	Mask       string
	Inline     bool
	String     bool
	Precision  int
	Format     string
	Unit       string
}

func (e *encoder) getStructInfo(rt reflect.Type) *structInfo {
	fieldCache.RLock()
	info, ok := fieldCache.m[rt]
	fieldCache.RUnlock()

	if ok {
		return info
	}

	fieldCache.Lock()
	defer fieldCache.Unlock()

	// Double-check
	if info, ok := fieldCache.m[rt]; ok {
		return info
	}

	info = &structInfo{}

	// Analyze struct fields
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)

		// Parse log tag
		if tagStr, ok := sf.Tag.Lookup("log"); ok {
			info.hasLogTag = true
			opts := e.parseFieldOptions(tagStr, sf)
			info.fields = append(info.fields, fieldInfo{
				index: i,
				name:  sf.Name,
				opts:  opts,
			})
			continue
		}

		// Parse json tag (for fallback)
		jsonTag := sf.Tag.Get("json")
		if jsonTag != "" {
			name := jsonTag
			opts := tagOpts("")
			if idx := strings.IndexByte(name, ','); idx >= 0 {
				name = name[:idx]
				opts = tagOpts(jsonTag[idx+1:])
			}
			info.fields = append(info.fields, fieldInfo{
				index:    i,
				jsonName: name,
				jsonOpts: opts,
			})
		}
	}

	fieldCache.m[rt] = info
	return info
}

func (e *encoder) parseFieldOptions(tag string, sf reflect.StructField) fieldOptions {
	opts := fieldOptions{
		Name: sf.Name,
	}

	tag = strings.TrimSpace(tag)
	if tag == "-" {
		opts.Name = "-"
		return opts
	}

	// Parse name and options
	if i := strings.IndexByte(tag, ','); i >= 0 {
		opts.Name = strings.TrimSpace(tag[:i])
		tag = strings.TrimSpace(tag[i+1:])
	} else {
		opts.Name = strings.TrimSpace(tag)
		return opts
	}

	// Parse options
	for _, seg := range strings.Split(tag, ",") {
		seg = strings.TrimSpace(seg)
		switch {
		case seg == "omitempty":
			opts.OmitEmpty = true
		case seg == "inline" && sf.Type.Kind() == reflect.Struct:
			opts.Inline = true
		case seg == "string":
			opts.String = true
		case strings.HasPrefix(seg, "ser="): // Changed from 'func=' to 'ser='
			opts.Serializer = strings.TrimPrefix(seg, "ser=")
		case strings.HasPrefix(seg, "mask="):
			opts.Mask = strings.TrimPrefix(seg, "mask=")
		case strings.HasPrefix(seg, "precision="):
			if p, err := strconv.Atoi(strings.TrimPrefix(seg, "precision=")); err == nil {
				opts.Precision = p
			}
		case strings.HasPrefix(seg, "format="):
			opts.Format = strings.TrimPrefix(seg, "format=")
		case strings.HasPrefix(seg, "unit="):
			opts.Unit = strings.TrimPrefix(seg, "unit=")
		}
	}

	return opts
}
