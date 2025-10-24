// Package slog provides field-level JSON logging, outputting only fields with the 'log' tag.
// Priority: Struct SLogger → Field SLogger → ser=xxx → Basic Type → Mask
package slog

import (
	"fmt"
	"reflect"
)

// ----- Error Type -----

type MarshalError struct {
	Type  reflect.Type
	Field string
	Err   error
}

func (e *MarshalError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("log: marshal field %s of type %s: %v", e.Field, e.Type, e.Err)
	}
	return fmt.Sprintf("log: marshal type %s: %v", e.Type, e.Err)
}

func (e *MarshalError) Unwrap() error {
	return e.Err
}

// ----- Configuration Options -----

// LogLevel represents the severity level of logging
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

type Options struct {
	DisableLoggerInterface bool     // Disable SLogger interface
	DisableJSONFallback    bool     // Disable JSON fallback
	OmitEmptyByDefault     bool     // Omit empty values by default
	MaskSensitive          bool     // Mask sensitive fields automatically
	Indent                 string   // JSON indent
	Prefix                 string   // JSON prefix
	EnableErrorFallback    bool     // Enable error fallback, output error info on serialization failure
	Level                  LogLevel // Log level for filtering
}

type Option func(*Options)

func WithOptions(opts Options) Option {
	return func(o *Options) { *o = opts }
}

func WithIndent(prefix, indent string) Option {
	return func(o *Options) {
		o.Prefix = prefix
		o.Indent = indent
	}
}

func WithMaskSensitive(mask bool) Option {
	return func(o *Options) { o.MaskSensitive = mask }
}

func WithErrorFallback(enable bool) Option {
	return func(o *Options) { o.EnableErrorFallback = enable }
}

func WithLevel(level LogLevel) Option {
	return func(o *Options) { o.Level = level }
}
