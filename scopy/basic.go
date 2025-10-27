package scopy

import (
	"reflect"
)

// copyBool copies boolean values
func (c *copier) copyBool(src, dst reflect.Value) error {
	dst.SetBool(src.Bool())
	return nil
}

// copyInt copies integer values
func (c *copier) copyInt(src, dst reflect.Value) error {
	dst.SetInt(src.Int())
	return nil
}

// copyUint copies unsigned integer values
func (c *copier) copyUint(src, dst reflect.Value) error {
	dst.SetUint(src.Uint())
	return nil
}

// copyFloat copies float values
func (c *copier) copyFloat(src, dst reflect.Value) error {
	dst.SetFloat(src.Float())
	return nil
}

// copyComplex copies complex values
func (c *copier) copyComplex(src, dst reflect.Value) error {
	dst.SetComplex(src.Complex())
	return nil
}

// copyString copies string values with optimization
func (c *copier) copyString(src, dst reflect.Value) error {
	// For strings, we can directly copy the value
	// Go strings are immutable, so this is safe
	srcStr := src.String()
	dst.SetString(srcStr)
	return nil
}

// copyUnsafePointer copies unsafe pointers
func (c *copier) copyUnsafePointer(src, dst reflect.Value) error {
	// Unsafe pointers are typically used for low-level operations
	// We copy them as-is, but this should be used with caution
	dst.SetPointer(src.UnsafePointer())
	return nil
}

// copyChan copies channels
func (c *copier) copyChan(src, dst reflect.Value) error {
	// Channels cannot be truly deep copied in Go
	// We create a new channel with the same capacity and direction
	if src.IsNil() {
		return nil
	}

	chanType := src.Type()
	capacity := src.Cap()

	// Create new channel with same capacity
	newChan := reflect.MakeChan(chanType, capacity)

	// Copy all values from source channel to destination
	// Note: This drains the source channel!
	for {
		select {
		case value, ok := <-src.Interface().(chan interface{}):
			if !ok {
				newChan.Close()
				dst.Set(newChan)
				return nil
			}
			// For deep copy, we need to copy the value
			copiedValue := reflect.New(reflect.TypeOf(value)).Elem()
			if err := c.copyValue(reflect.ValueOf(value), copiedValue); err != nil {
				return err
			}
			newChan.Send(copiedValue)
		default:
			dst.Set(newChan)
			return nil
		}
	}
}

// copyFunc copies function values
func (c *copier) copyFunc(src, dst reflect.Value) error {
	// Functions in Go are reference types and cannot be deep copied
	// We just copy the reference
	dst.Set(src)
	return nil
}
