package scopy

import (
	"reflect"
)

// copyArray copies array values
func (c *copier) copyArray(src, dst reflect.Value) error {
	if src.Len() != dst.Len() {
		return &copyError{Type: src.Type(), Message: "array length mismatch"}
	}

	for i := 0; i < src.Len(); i++ {
		if err := c.copyValue(src.Index(i), dst.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

// copySlice copies slice values with optimization
func (c *copier) copySlice(src, dst reflect.Value) error {
	if src.IsNil() {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	srcLen := src.Len()
	if srcLen == 0 {
		dst.Set(reflect.MakeSlice(dst.Type(), 0, 0))
		return nil
	}

	// Create new slice with same length and capacity
	newSlice := reflect.MakeSlice(dst.Type(), srcLen, src.Cap())

	// Copy elements
	for i := 0; i < srcLen; i++ {
		if err := c.copyValue(src.Index(i), newSlice.Index(i)); err != nil {
			return err
		}
	}

	dst.Set(newSlice)
	return nil
}

// copyMap copies map values with optimization
func (c *copier) copyMap(src, dst reflect.Value) error {
	if src.IsNil() {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	srcLen := src.Len()
	if srcLen == 0 {
		dst.Set(reflect.MakeMapWithSize(dst.Type(), 0))
		return nil
	}

	// Create new map with same size
	mapType := dst.Type()
	newMap := reflect.MakeMapWithSize(mapType, srcLen)

	// Copy key-value pairs
	for _, key := range src.MapKeys() {
		value := src.MapIndex(key)

		// Copy key
		newKey := reflect.New(key.Type()).Elem()
		if err := c.copyValue(key, newKey); err != nil {
			return err
		}

		// Copy value
		newValue := reflect.New(value.Type()).Elem()
		if err := c.copyValue(value, newValue); err != nil {
			return err
		}

		newMap.SetMapIndex(newKey, newValue)
	}

	dst.Set(newMap)
	return nil
}

// copyStruct copies struct values with field-by-field copying
func (c *copier) copyStruct(src, dst reflect.Value) error {
	srcType := src.Type()

	// Check cache for struct copy function
	if c.opts.EnableCache {
		if cachedFunc, ok := c.cache.Load(srcType); ok {
			if copyFunc, ok := cachedFunc.(func(*copier, reflect.Value, reflect.Value) error); ok {
				return copyFunc(c, src, dst)
			}
		}
	}

	// Create optimized copy function for this struct type
	var copyFunc func(*copier, reflect.Value, reflect.Value) error

	if c.opts.EnableCache {
		copyFunc = c.generateStructCopyFunc(srcType)
		c.cache.Store(srcType, copyFunc)
	} else {
		copyFunc = c.generateStructCopyFunc(srcType)
	}

	return copyFunc(c, src, dst)
}

// generateStructCopyFunc generates an optimized copy function for a specific struct type
func (c *copier) generateStructCopyFunc(structType reflect.Type) func(*copier, reflect.Value, reflect.Value) error {
	return func(c *copier, src, dst reflect.Value) error {
		// Ensure both src and dst are structs
		if src.Kind() != reflect.Struct || dst.Kind() != reflect.Struct {
			return &copyError{
				Type:    src.Type(),
				Message: "expected struct types",
			}
		}

		numFields := src.NumField()

		for i := 0; i < numFields; i++ {
			srcField := src.Field(i)
			dstField := dst.Field(i)

			// Skip unexported fields
			if !dstField.CanSet() {
				continue
			}

			// Skip zero values if option is enabled
			if c.opts.SkipZeroValues && isZero(srcField) {
				continue
			}

			if err := c.copyValue(srcField, dstField); err != nil {
				return &copyError{
					Type:    src.Type(),
					Field:   src.Type().Field(i).Name,
					Message: err.Error(),
				}
			}
		}
		return nil
	}
}

// copyPtr copies pointer values with cycle detection
func (c *copier) copyPtr(src, dst reflect.Value) error {
	if src.IsNil() {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	// Check for cycles
	ptrKey := src.Pointer()
	if visited, ok := c.visited.Load(ptrKey); ok {
		if visitedValue, ok := visited.(reflect.Value); ok {
			dst.Set(visitedValue)
			return nil
		}
	}

	// Create new pointer value
	newPtr := reflect.New(src.Elem().Type())

	// Store in visited map to handle cycles
	c.visited.Store(ptrKey, newPtr)

	// Copy the pointed value
	if err := c.copyValue(src.Elem(), newPtr.Elem()); err != nil {
		return err
	}

	dst.Set(newPtr)
	return nil
}

// copyInterface copies interface values
func (c *copier) copyInterface(src, dst reflect.Value) error {
	if src.IsNil() {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	// Get the concrete value
	srcElem := src.Elem()

	// Create new interface value
	newInterface := reflect.New(srcElem.Type()).Elem()

	// Copy the concrete value
	if err := c.copyValue(srcElem, newInterface); err != nil {
		return err
	}

	dst.Set(newInterface)
	return nil
}

// copyError represents a copy error with type information
type copyError struct {
	Type    reflect.Type
	Field   string
	Message string
}

func (e *copyError) Error() string {
	if e.Field != "" {
		return "scopy: failed to copy " + e.Type.String() + "." + e.Field + ": " + e.Message
	}
	return "scopy: failed to copy " + e.Type.String() + ": " + e.Message
}