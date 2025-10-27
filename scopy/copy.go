// Package scopy provides high-performance deep copy functionality for Go types.
// It supports all basic types, complex types (slices, maps, structs), pointers,
// interfaces, and custom types with optimal performance through reflection and
// code generation techniques.
package scopy

import (
	"fmt"
	"reflect"
	"sync"
)

// Copier is the main interface for deep copy operations
type Copier interface {
	// Copy performs a deep copy of the given value
	Copy(src interface{}) (interface{}, error)

	// CopyTo performs a deep copy from src to dst
	CopyTo(src, dst interface{}) error
}

// Options configures the copier behavior
type Options struct {
	// MaxDepth limits the maximum recursion depth to prevent infinite loops
	MaxDepth int

	// EnableCache enables caching for struct types to improve performance
	EnableCache bool

	// SkipZeroValues skips copying zero values for performance
	SkipZeroValues bool

	// CustomCopyers allows registration of custom copy functions for specific types
	CustomCopyers map[reflect.Type]CopyFunc
}

// CopyFunc is a function that can copy a specific type
type CopyFunc func(src, dst reflect.Value) error

// DefaultOptions returns default copier options
func DefaultOptions() *Options {
	return &Options{
		MaxDepth:       100,
		EnableCache:    true,
		SkipZeroValues: false,
		CustomCopyers:  make(map[reflect.Type]CopyFunc),
	}
}

// copier is the main implementation
type copier struct {
	opts      *Options
	cache     *sync.Map // type cache for performance
	depth     int       // current recursion depth
	visited   *sync.Map // track visited pointers to handle cycles
}

// New creates a new copier with the given options
func New(opts *Options) Copier {
	if opts == nil {
		opts = DefaultOptions()
	}

	return &copier{
		opts:    opts,
		cache:   &sync.Map{},
		visited: &sync.Map{},
	}
}

// Copy performs a deep copy of the given value
func (c *copier) Copy(src interface{}) (interface{}, error) {
	if src == nil {
		return nil, nil
	}

	srcValue := reflect.ValueOf(src)
	dstValue := reflect.New(srcValue.Type()).Elem()

	c.depth = 0
	c.visited = &sync.Map{}

	err := c.copyValue(srcValue, dstValue)
	if err != nil {
		return nil, err
	}

	return dstValue.Interface(), nil
}

// CopyTo performs a deep copy from src to dst
func (c *copier) CopyTo(src, dst interface{}) error {
	if src == nil || dst == nil {
		return fmt.Errorf("src and dst cannot be nil")
	}

	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	if dstValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be a pointer")
	}

	c.depth = 0
	c.visited = &sync.Map{}

	return c.copyValue(srcValue, dstValue.Elem())
}

// copyValue is the main copy function that handles all types
func (c *copier) copyValue(src, dst reflect.Value) error {
	if !src.IsValid() {
		return nil
	}

	if c.opts.SkipZeroValues && isZero(src) {
		return nil
	}

	// Check max depth to prevent stack overflow
	if c.depth > c.opts.MaxDepth {
		return fmt.Errorf("max depth exceeded: %d", c.opts.MaxDepth)
	}
	c.depth++
	defer func() { c.depth-- }()

	// Handle nil values
	if src.Kind() == reflect.Ptr && src.IsNil() {
		return nil
	}

	// Check for custom copier
	if c.opts.EnableCache {
		if copyFunc, ok := c.opts.CustomCopyers[src.Type()]; ok {
			return copyFunc(src, dst)
		}
	}

	// Handle different types
	switch src.Kind() {
	case reflect.Bool:
		return c.copyBool(src, dst)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return c.copyInt(src, dst)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return c.copyUint(src, dst)
	case reflect.Float32, reflect.Float64:
		return c.copyFloat(src, dst)
	case reflect.Complex64, reflect.Complex128:
		return c.copyComplex(src, dst)
	case reflect.String:
		return c.copyString(src, dst)
	case reflect.Array:
		return c.copyArray(src, dst)
	case reflect.Slice:
		return c.copySlice(src, dst)
	case reflect.Map:
		return c.copyMap(src, dst)
	case reflect.Struct:
		return c.copyStruct(src, dst)
	case reflect.Ptr:
		return c.copyPtr(src, dst)
	case reflect.Interface:
		return c.copyInterface(src, dst)
	case reflect.Chan:
		return c.copyChan(src, dst)
	case reflect.Func:
		return c.copyFunc(src, dst)
	case reflect.UnsafePointer:
		return c.copyUnsafePointer(src, dst)
	default:
		return fmt.Errorf("unsupported type: %v", src.Type())
	}
}

// Helper function to check if a value is zero
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Struct:
		return false // Can't determine if struct is zero without reflection
	default:
		return false
	}
}