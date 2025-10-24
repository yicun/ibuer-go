// Package log provides field-level JSON logging, outputting only fields with the 'log' tag.
// Priority: Field log:ser=xxx → Field Struct Logger → Basic Type → Mask
package log

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
)

// ----- Encoder Implementation -----

type encoder struct {
	out     any
	visited map[uintptr]bool
	opts    *Options
}

func newEncoder() *encoder {
	enc := encoderPool.Get().(*encoder)
	// Reset state
	enc.out = nil
	enc.opts = nil
	for k := range enc.visited {
		delete(enc.visited, k)
	}
	return enc
}

func releaseEncoder(enc *encoder) {
	// Clear encoder state to prevent memory leaks
	enc.out = nil
	enc.opts = nil
	// Clear visited map more efficiently
	if len(enc.visited) > 0 {
		// Only clear if there are entries
		for k := range enc.visited {
			delete(enc.visited, k)
		}
	}
	encoderPool.Put(enc)
}

func (e *encoder) encode(v any) error {
	// Check conditional logging
	if cl, ok := v.(ConditionalLogger); ok && !cl.ShouldLog() {
		e.out = nil
		return nil
	}

	if !e.opts.DisableLoggerInterface {
		if lg, ok := v.(Logger); ok {
			b, err := lg.MarshalLog()
			if err != nil {
				return err
			}
			e.out = json.RawMessage(b)
			return nil
		}
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Ptr && rv.CanAddr() {
			if lg, ok := rv.Addr().Interface().(Logger); ok {
				b, err := lg.MarshalLog()
				if err != nil {
					return err
				}
				e.out = json.RawMessage(b)
				return nil
			}
		}
	}
	return e.encodeReflect(reflect.ValueOf(v))
}

func (e *encoder) encodeReflect(rv reflect.Value) error {
	// Handle pointers and circular references
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			e.out = nil
			return nil
		}

		ptr := rv.Pointer()
		if e.visited[ptr] {
			return &MarshalError{
				Type: rv.Type(),
				Err:  fmt.Errorf("cyclic reference detected"),
			}
		}
		e.visited[ptr] = true
		defer delete(e.visited, ptr)

		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Struct:
		return e.encodeStruct(rv)
	case reflect.Slice, reflect.Array:
		if rv.Len() == 0 && e.opts.OmitEmptyByDefault {
			e.out = nil
			return nil
		}
		arr := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			var sub encoder
			sub.visited = e.visited
			sub.opts = e.opts
			if err := sub.encodeReflect(rv.Index(i)); err != nil {
				return err
			}
			if sub.out != nil {
				arr = append(arr, sub.out)
			}
		}
		e.out = arr
		return nil
	case reflect.Map:
		if rv.Len() == 0 && e.opts.OmitEmptyByDefault {
			e.out = nil
			return nil
		}
		m := make(map[string]any)
		for _, k := range rv.MapKeys() {
			if k.Kind() != reflect.String {
				continue
			}
			var sub encoder
			sub.visited = e.visited
			sub.opts = e.opts
			if err := sub.encodeReflect(rv.MapIndex(k)); err != nil {
				return err
			}
			if sub.out != nil {
				m[k.String()] = sub.out
			}
		}
		e.out = m
		return nil
	case reflect.String, reflect.Bool, reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		e.out = rv.Interface()
		return nil
	case reflect.Interface:
		if rv.IsNil() {
			e.out = nil
			return nil
		}
		return e.encodeReflect(rv.Elem())
	default:
		e.out = nil
		return nil
	}
}

func (e *encoder) encodeStruct(rv reflect.Value) error {
	rt := rv.Type()

	// 1. Struct Logger (if not disabled)
	if !e.opts.DisableLoggerInterface {
		if lg, ok := rv.Interface().(Logger); ok {
			b, err := lg.MarshalLog()
			if err != nil {
				return &MarshalError{Type: rt, Err: err}
			}
			e.out = json.RawMessage(b)
			return nil
		}
		if rv.CanAddr() {
			if lg, ok := rv.Addr().Interface().(Logger); ok {
				b, err := lg.MarshalLog()
				if err != nil {
					return &MarshalError{Type: rt, Err: err}
				}
				e.out = json.RawMessage(b)
				return nil
			}
		}
	}

	// Get cached struct info
	info := e.getStructInfo(rt)

	// 2. Has log tags → process only these fields
	if info.hasLogTag {
		m := make(map[string]any)
		for _, fi := range info.fields {
			fv := rv.Field(fi.index)
			sf := rt.Field(fi.index)

			if err := e.encodeField(sf, fv, fi, m); err != nil {
				return &MarshalError{Type: rt, Field: sf.Name, Err: err}
			}
		}
		e.out = m
		return nil
	}

	// 3. json.Marshaler (if fallback not disabled)
	if !e.opts.DisableJSONFallback {
		if mj, ok := rv.Interface().(json.Marshaler); ok {
			b, err := mj.MarshalJSON()
			if err != nil {
				return &MarshalError{Type: rt, Err: err}
			}
			e.out = json.RawMessage(b)
			return nil
		}
		if rv.CanAddr() {
			if mj, ok := rv.Addr().Interface().(json.Marshaler); ok {
				b, err := mj.MarshalJSON()
				if err != nil {
					return &MarshalError{Type: rt, Err: err}
				}
				e.out = json.RawMessage(b)
				return nil
			}
		}
	}

	// 4. Fallback to JSON tags
	if !e.opts.DisableJSONFallback {
		m := make(map[string]any)
		for _, fi := range info.fields {
			if fi.jsonName == "" || fi.jsonName == "-" {
				continue
			}
			fv := rv.Field(fi.index)
			sf := rt.Field(fi.index)

			if fi.jsonOpts.Contains("omitempty") && isEmpty(fv) {
				continue
			}

			var sub encoder
			sub.visited = e.visited
			sub.opts = e.opts
			if err := sub.encodeReflect(fv); err != nil {
				return &MarshalError{Type: rt, Field: sf.Name, Err: err}
			}
			if sub.out != nil {
				m[fi.jsonName] = sub.out
			}
		}
		e.out = m
		return nil
	}

	// 5. Skip completely
	e.out = nil
	return nil
}

func (e *encoder) encodeField(sf reflect.StructField, fv reflect.Value, fi fieldInfo, m map[string]any) error {
	// Check if field should be ignored (log:"-")
	if fi.opts.Name == "-" {
		return nil
	}

	// Check conditional logging
	if cl, ok := fv.Interface().(ConditionalLogger); ok {
		if !cl.ShouldLog() {
			return nil
		}
		// If ShouldLog() returns true, serialize the entire struct using JSON marshaling
		// This preserves all struct fields, not just those with log tags
		var sub encoder
		sub.visited = e.visited
		sub.opts = e.opts
		if err := sub.encodeReflect(fv); err != nil {
			// If error fallback is enabled, output error info
			if e.opts.EnableErrorFallback {
				errorStr := e.createErrorString(sf.Name, fv, err)
				m[fi.opts.Name] = errorStr
				return nil
			}
			return err
		}
		if sub.out != nil {
			m[fi.opts.Name] = sub.out
		}
		return nil
	}

	// Priority: Field log:ser=xxx → Field Struct Logger → Basic Type → Mask

	// Early check for omitempty (both field-level and global)
	if (fi.opts.OmitEmpty || e.opts.OmitEmptyByDefault) && isEmpty(fv) {
		return nil
	}

	// 1. Field log:"ser=xxx" (field-level custom serializer - highest priority)
	if handled, err := e.encodeWithSerializer(fv, sf, fi, m); handled || err != nil {
		return err
	}

	// 2. Field Struct Logger (field with Logger interface)
	if !e.opts.DisableLoggerInterface {
		if lg, ok := fv.Interface().(Logger); ok && fv.CanInterface() {
			b, err := lg.MarshalLog()
			if err != nil {
				// If error fallback is enabled, output error info
				if e.opts.EnableErrorFallback {
					errorStr := e.createErrorString(sf.Name, fv, err)
					m[fi.opts.Name] = errorStr
					return nil
				}
				return err
			}
			// Note: Logger interface results are not subject to omitempty
			m[fi.opts.Name] = json.RawMessage(b)
			return nil
		}
	}

	// 3. Basic Type serialization with post-processing
	return e.encodeBasic(fv, sf, fi, m)
}

// createErrorString creates a string containing field name, raw value, and error info.
func (e *encoder) createErrorString(fieldName string, fv reflect.Value, err error) string {
	// Safely get string representation of field value
	valueStr := e.safeValueToString(fv)

	// Check if field name or value contains sensitive information
	if e.containsSensitiveData(fieldName, valueStr) {
		valueStr = "<sensitive>"
	}

	// Build error info string
	errorInfo := fmt.Sprintf("FIELD_SERIALIZE_ERROR{field:%s, value:%s, error:%s}",
		fieldName, valueStr, err.Error())

	return errorInfo
}

// encodeWithLogger handles encoding using the Logger interface (highest priority)
func (e *encoder) encodeWithLogger(fv reflect.Value, sf reflect.StructField, fi fieldInfo, m map[string]any) (bool, error) {
	if e.opts.DisableLoggerInterface {
		return false, nil
	}

	if lg, ok := fv.Interface().(Logger); ok && fv.CanInterface() {
		b, err := lg.MarshalLog()
		if err != nil {
			// If error fallback is enabled, output error info
			if e.opts.EnableErrorFallback {
				errorStr := e.createErrorString(sf.Name, fv, err)
				m[fi.opts.Name] = errorStr
				return true, nil
			}
			return true, err
		}
		// Note: Logger interface results are not subject to omitempty
		m[fi.opts.Name] = json.RawMessage(b)
		return true, nil
	}
	return false, nil
}

// encodeWithSerializer handles encoding using custom serializers
func (e *encoder) encodeWithSerializer(fv reflect.Value, sf reflect.StructField, fi fieldInfo, m map[string]any) (bool, error) {
	if fi.opts.Serializer == "" {
		return false, nil
	}

	// Try to get serializer from main registry first, then lazy registry
	var fn SerializerFunc
	var ok bool

	// Check main registry first
	if fn, ok = getSer(fi.opts.Serializer); !ok {
		// Check lazy registry
		fn, ok = getLazySerializer(fi.opts.Serializer)
	}

	if !ok {
		// Serializer not found - handle according to error fallback setting
		if e.opts.EnableErrorFallback {
			errorStr := e.createErrorString(sf.Name, fv, fmt.Errorf("serializer '%s' not found", fi.opts.Serializer))
			m[fi.opts.Name] = errorStr
			return true, nil
		}
		return true, &MarshalError{Type: fv.Type(), Field: sf.Name, Err: fmt.Errorf("serializer '%s' not found", fi.opts.Serializer)}
	}

	// Serializer found, execute it
	b, err := fn(fv.Interface())
	if err != nil {
		// Serializer error - handle according to error fallback setting
		if e.opts.EnableErrorFallback {
			errorStr := e.createErrorString(sf.Name, fv, err)
			m[fi.opts.Name] = errorStr
			return true, nil
		}
		return true, &MarshalError{Type: fv.Type(), Field: sf.Name, Err: err}
	}

	// Note: Custom serializer results are not subject to omitempty
	m[fi.opts.Name] = json.RawMessage(b)
	return true, nil
}

// encodeBasic handles basic type serialization and post-processing
func (e *encoder) encodeBasic(fv reflect.Value, sf reflect.StructField, fi fieldInfo, m map[string]any) error {
	// Inline field handling (for struct types)
	if fi.opts.Inline && fv.Kind() == reflect.Struct {
		return e.encodeInlineField(fv, sf, fi, m)
	}

	var sub encoder
	sub.visited = e.visited
	sub.opts = e.opts
	if err := sub.encodeReflect(fv); err != nil {
		// If error fallback is enabled, output error info
		if e.opts.EnableErrorFallback {
			errorStr := e.createErrorString(sf.Name, fv, err)
			m[fi.opts.Name] = errorStr
			return nil
		}
		return err
	}

	// Apply post-processing (mask, precision, string formatting)
	e.applyFieldPostProcessing(fi, sf.Name, fv, sub.out, m)
	return nil
}

// encodeInlineField handles inline field processing for struct types
func (e *encoder) encodeInlineField(fv reflect.Value, sf reflect.StructField, fi fieldInfo, m map[string]any) error {
	var sub encoder
	sub.visited = e.visited
	sub.opts = e.opts
	if err := sub.encodeReflect(fv); err != nil {
		// If error fallback is enabled, output error info
		if e.opts.EnableErrorFallback {
			errorStr := e.createErrorString(sf.Name, fv, err)
			m[fi.opts.Name] = errorStr
			return nil
		}
		return err
	}
	if subMap, ok := sub.out.(map[string]any); ok {
		for k, v := range subMap {
			m[k] = v
		}
	}
	return nil
}

// containsSensitiveData checks if field name or value contains sensitive information
func (e *encoder) containsSensitiveData(fieldName, value string) bool {
	// Check field name for sensitive keywords
	sensitiveFieldNames := []string{
		"password", "passwd", "pwd", "secret", "token", "key",
		"credential", "auth", "private", "credit", "ssn",
		"phone", "email", "address", "card", "bank",
	}

	lowerFieldName := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFieldNames {
		if strings.Contains(lowerFieldName, sensitive) {
			return true
		}
	}

	// Check value patterns for sensitive data
	// Email pattern
	if strings.Contains(value, "@") && strings.Contains(value, ".") {
		return true
	}

	// Phone number pattern (simple check for 11 digits)
	if matched, _ := regexp.MatchString(`\d{11}`, value); matched {
		return true
	}

	// Credit card pattern (16 digits with possible spaces/dashes)
	if matched, _ := regexp.MatchString(`\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}`, value); matched {
		return true
	}

	return false
}

// applyFieldPostProcessing applies mask, precision, and string formatting to field values
func (e *encoder) applyFieldPostProcessing(fi fieldInfo, fieldName string, fv reflect.Value, val any, m map[string]any) {
	if val == nil {
		return
	}

	// Masking
	if (e.opts.MaskSensitive || fi.opts.Mask != "") && val != nil {
		if s, ok := val.(string); ok {
			maskName := fi.opts.Mask
			if maskName == "" {
				maskName = defaultMK
			}
			val = getMask(maskName)(s)
		}
	}

	// Precision for floats
	if fi.opts.Precision > 0 {
		if f, ok := val.(float32); ok {
			multiplier := float32(math.Pow(10, float64(fi.opts.Precision)))
			val = math.Round(float64(f)*float64(multiplier)) / float64(multiplier)
		} else if f, ok := val.(float64); ok {
			multiplier := math.Pow(10, float64(fi.opts.Precision))
			val = math.Round(f*multiplier) / multiplier
		}
	}

	// Force string format
	if fi.opts.String && val != nil {
		val = fmt.Sprintf("%v", val)
	}

	if val != nil {
		m[fi.opts.Name] = val
	}
}

// safeValueToString safely converts field value to string, avoiding panic.
func (e *encoder) safeValueToString(fv reflect.Value) (str string) {
	defer func() {
		if r := recover(); r != nil {
			str = fmt.Sprintf("<panic:%v>", r)
		}
	}()

	if !fv.IsValid() {
		return "<invalid>"
	}

	switch fv.Kind() {
	case reflect.String:
		return fv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", fv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", fv.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", fv.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", fv.Bool())
	case reflect.Ptr, reflect.Interface:
		if fv.IsNil() {
			return "<nil>"
		}
		return e.safeValueToString(fv.Elem())
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("%s[len=%d]", fv.Type().String(), fv.Len())
	case reflect.Map:
		return fmt.Sprintf("%s[len=%d]", fv.Type().String(), fv.Len())
	case reflect.Struct:
		return fmt.Sprintf("%s{...}", fv.Type().String())
	default:
		return fmt.Sprintf("%v", fv.Interface())
	}
}
