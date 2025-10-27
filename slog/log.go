// Package slog provides field-level JSON logging, outputting only fields with the 'log' tag.
// Priority: Struct SLogger → ser=xxx → Field SLogger → Basic Type → Mask
package slog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"
)

// ----- Interfaces -----

type (
	SLogger interface {
		MarshalLog() ([]byte, error)
	}

	SConditionalLogger interface {
		ShouldLog() bool
	}

	MaskFunc       func(string) string
	SerializerFunc func(any) ([]byte, error)
)

// ----- Public API -----

// Marshal serializes the value using default options.
func Marshal(v any) ([]byte, error) {
	return MarshalWithOpts(v)
}

// MarshalWithOpts serializes the value with custom options.
func MarshalWithOpts(v any, opts ...Option) ([]byte, error) {
	enc := newEncoder()
	defer releaseEncoder(enc)

	options := &Options{
		MaskSensitive:       false,
		EnableErrorFallback: true,
		Level:               INFO, // Default log level
	}
	for _, opt := range opts {
		opt(options)
	}
	enc.opts = options

	if err := enc.encode(v); err != nil {
		return nil, err
	}

	if enc.out == nil {
		return []byte("null"), nil
	}

	var buf bytes.Buffer
	jsonEnc := json.NewEncoder(&buf)
	if options.Indent != "" {
		jsonEnc.SetIndent(options.Prefix, options.Indent)
	}

	if err := jsonEnc.Encode(enc.out); err != nil {
		return nil, err
	}

	// Remove trailing newline added by json.Encoder
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

// MarshalWithContext serializes the value with context-aware options.
func MarshalWithContext(ctx context.Context, v any, opts ...Option) ([]byte, error) {
	// Handle nil context gracefully
	if ctx == nil {
		return MarshalWithOpts(v, opts...)
	}

	// Extract trace information from context if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if traceStr, ok := traceID.(string); ok {
			// Add trace ID to the options or handle it appropriately
			opts = append(opts, func(o *Options) {
				// Store trace ID in options for potential use by serializers
				if o.Prefix == "" {
					o.Prefix = traceStr
				}
			})
		}
	}

	return MarshalWithOpts(v, opts...)
}

// MarshalTo writes the serialized value to the writer.
func MarshalTo(w io.Writer, v any, opts ...Option) error {
	data, err := MarshalWithOpts(v, opts...)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// RegisterMask registers a mask function.
func RegisterMask(name string, fn MaskFunc) {
	if name == "" {
		fmt.Printf("log: mask name is nil, return\n")
		return
	}
	// Use LoadOrStore for atomic operation
	if _, loaded := maskReg.LoadOrStore(name, fn); loaded {
		// Overwrite existing mask atomically
		maskReg.Store(name, fn)
		fmt.Printf("log: mask %q already registered, overwritten\n", name)
	}
}

// RegisterSerializer registers a serializer function.
func RegisterSerializer(name string, fn SerializerFunc) {
	if name == "" {
		fmt.Printf("log: serializer name is nil, return\n")
		return
	}
	// Use LoadOrStore for atomic operation
	if _, loaded := serReg.LoadOrStore(name, fn); loaded {
		// Overwrite existing serializer atomically
		serReg.Store(name, fn)
		fmt.Printf("log: serializer %q already registered, overwritten\n", name)
	}
}

// getMask retrieves a mask function.
func getMask(name string) MaskFunc {
	if name == "" {
		name = defaultMK
	}
	if v, ok := maskReg.Load(name); ok {
		return v.(MaskFunc)
	}
	return defaultMask
}

// getSer retrieves a serializer function.
func getSer(name string) (SerializerFunc, bool) {
	v, ok := serReg.Load(name)
	if !ok {
		return nil, false
	}
	return v.(SerializerFunc), true
}

// ----- Global Registry -----
var (
	maskReg   sync.Map
	serReg    sync.Map
	defaultMK = "default"

	// Encoder pool
	encoderPool = sync.Pool{
		New: func() interface{} {
			return &encoder{
				visited: make(map[uintptr]bool),
			}
		},
	}

	// Struct info cache
	fieldCache = &structCache{
		m: make(map[reflect.Type]*structInfo),
	}
)

// ----- Initialization -----
func init() {
	// Register default masks
	RegisterMask("phone", func(s string) string {
		if len(s) == 11 {
			return s[:3] + "****" + s[7:]
		}
		return "***"
	})
	RegisterMask("email", func(s string) string {
		if idx := strings.IndexByte(s, '@'); idx > 0 {
			if idx > 3 {
				return s[:3] + "***" + s[idx:]
			}
			return s[:1] + "***" + s[idx:]
		}
		return "***"
	})

	// Register default serializers with lazy loading
	RegisterCurrencyFormattedSerializer()
	RegisterTimeFormattedSerializer()
	RegisterDurationFormattedSerializer()

	// Register custom time serializers (these are simple, keep as immediate)
	RegisterTimeSerializerWithLayout("time_short_date", "2006-01-02")
	RegisterTimeSerializerWithLayout("time_long_date", "2006年01月02日")
	RegisterTimeSerializerWithLayout("time_filename", "20060102_150405")
	RegisterTimeSerializerWithLayout("time_log", "2006/01/02 15:04:05.000")
	RegisterDurationSerializerWithPrecision("duration_sec_2", time.Second, 2)
	RegisterDurationSerializerWithPrecision("duration_sec_3", time.Second, 3)
	RegisterDurationSerializerWithPrecision("duration_ms_1", time.Millisecond, 1)
	RegisterDurationSerializerWithPrecision("duration_min_2", time.Minute, 2)
}
