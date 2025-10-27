package scopy

import (
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

// TypeInfo holds cached type information for performance optimization
type TypeInfo struct {
	Type         reflect.Type
	Size         uintptr
	IsPrimitive  bool
	CopyFunc     func(*copier, reflect.Value, reflect.Value) error
	FieldOffsets []uintptr // For struct types
}

// typeCache is a thread-safe cache for type information
type typeCache struct {
	mu    sync.RWMutex
	cache map[reflect.Type]*TypeInfo
	hits  uint64
	miss  uint64
}

// newTypeCache creates a new type cache
func newTypeCache() *typeCache {
	return &typeCache{
		cache: make(map[reflect.Type]*TypeInfo),
	}
}

// get retrieves type information from cache
func (tc *typeCache) get(typ reflect.Type) (*TypeInfo, bool) {
	tc.mu.RLock()
	info, ok := tc.cache[typ]
	tc.mu.RUnlock()

	if ok {
		atomic.AddUint64(&tc.hits, 1)
		return info, true
	}

	atomic.AddUint64(&tc.miss, 1)
	return nil, false
}

// set stores type information in cache
func (tc *typeCache) set(typ reflect.Type, info *TypeInfo) {
	tc.mu.Lock()
	tc.cache[typ] = info
	tc.mu.Unlock()
}

// stats returns cache statistics
func (tc *typeCache) stats() (hits, misses uint64) {
	return atomic.LoadUint64(&tc.hits), atomic.LoadUint64(&tc.miss)
}

// analyzeType analyzes a type and returns optimization information
func analyzeType(typ reflect.Type) *TypeInfo {
	info := &TypeInfo{
		Type:        typ,
		Size:        typ.Size(),
		IsPrimitive: isPrimitiveType(typ),
	}

	switch typ.Kind() {
	case reflect.Struct:
		info.FieldOffsets = calculateFieldOffsets(typ)
		info.CopyFunc = generateOptimizedStructCopy(typ)
	case reflect.Slice:
		info.CopyFunc = generateOptimizedSliceCopy(typ)
	case reflect.Map:
		info.CopyFunc = generateOptimizedMapCopy(typ)
	case reflect.Ptr:
		info.CopyFunc = generateOptimizedPtrCopy(typ)
	default:
		if info.IsPrimitive {
			info.CopyFunc = generatePrimitiveCopy(typ)
		}
	}

	return info
}

// isPrimitiveType checks if a type is a primitive type that can be copied directly
func isPrimitiveType(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	default:
		return false
	}
}

// calculateFieldOffsets calculates field offsets for struct types
func calculateFieldOffsets(typ reflect.Type) []uintptr {
	if typ.Kind() != reflect.Struct {
		return nil
	}

	numFields := typ.NumField()
	offsets := make([]uintptr, numFields)

	for i := 0; i < numFields; i++ {
		field := typ.Field(i)
		offsets[i] = field.Offset
	}

	return offsets
}

// generateOptimizedStructCopy generates an optimized copy function for struct types
func generateOptimizedStructCopy(typ reflect.Type) func(*copier, reflect.Value, reflect.Value) error {
	return func(c *copier, src, dst reflect.Value) error {
		numFields := src.NumField()

		for i := 0; i < numFields; i++ {
			srcField := src.Field(i)
			dstField := dst.Field(i)

			if !dstField.CanSet() {
				continue
			}

			if c.opts.SkipZeroValues && isZero(srcField) {
				continue
			}

			// Use optimized copy for primitive types
			if isPrimitiveType(srcField.Type()) {
				copyPrimitiveValue(srcField, dstField)
			} else {
				if err := c.copyValue(srcField, dstField); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// generateOptimizedSliceCopy generates an optimized copy function for slice types
func generateOptimizedSliceCopy(typ reflect.Type) func(*copier, reflect.Value, reflect.Value) error {
	elemType := typ.Elem()
	isPrimitive := isPrimitiveType(elemType)

	return func(c *copier, src, dst reflect.Value) error {
		if src.IsNil() {
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		}

		srcLen := src.Len()
		if srcLen == 0 {
			dst.Set(reflect.MakeSlice(dst.Type(), 0, 0))
			return nil
		}

		newSlice := reflect.MakeSlice(dst.Type(), srcLen, src.Cap())

		if isPrimitive {
			// Use bulk copy for primitive types
			copyPrimitiveSlice(src, newSlice)
		} else {
			// Element-by-element copy for complex types
			for i := 0; i < srcLen; i++ {
				if err := c.copyValue(src.Index(i), newSlice.Index(i)); err != nil {
					return err
				}
			}
		}

		dst.Set(newSlice)
		return nil
	}
}

// generateOptimizedMapCopy generates an optimized copy function for map types
func generateOptimizedMapCopy(typ reflect.Type) func(*copier, reflect.Value, reflect.Value) error {
	keyType := typ.Key()
	elemType := typ.Elem()
	keyIsPrimitive := isPrimitiveType(keyType)
	elemIsPrimitive := isPrimitiveType(elemType)

	return func(c *copier, src, dst reflect.Value) error {
		if src.IsNil() {
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		}

		srcLen := src.Len()
		if srcLen == 0 {
			dst.Set(reflect.MakeMapWithSize(dst.Type(), 0))
			return nil
		}

		newMap := reflect.MakeMapWithSize(dst.Type(), srcLen)

		for _, key := range src.MapKeys() {
			value := src.MapIndex(key)

			var newKey, newValue reflect.Value

			if keyIsPrimitive {
				newKey = reflect.New(keyType).Elem()
				copyPrimitiveValue(key, newKey)
			} else {
				newKey = reflect.New(keyType).Elem()
				if err := c.copyValue(key, newKey); err != nil {
					return err
				}
			}

			if elemIsPrimitive {
				newValue = reflect.New(elemType).Elem()
				copyPrimitiveValue(value, newValue)
			} else {
				newValue = reflect.New(elemType).Elem()
				if err := c.copyValue(value, newValue); err != nil {
					return err
				}
			}

			newMap.SetMapIndex(newKey, newValue)
		}

		dst.Set(newMap)
		return nil
	}
}

// generateOptimizedPtrCopy generates an optimized copy function for pointer types
func generateOptimizedPtrCopy(typ reflect.Type) func(*copier, reflect.Value, reflect.Value) error {
	elemType := typ.Elem()
	elemIsPrimitive := isPrimitiveType(elemType)

	return func(c *copier, src, dst reflect.Value) error {
		if src.IsNil() {
			dst.Set(reflect.Zero(dst.Type()))
			return nil
		}

		ptrKey := src.Pointer()
		if visited, ok := c.visited.Load(ptrKey); ok {
			if visitedValue, ok := visited.(reflect.Value); ok {
				dst.Set(visitedValue)
				return nil
			}
		}

		newPtr := reflect.New(elemType)
		c.visited.Store(ptrKey, newPtr)

		if elemIsPrimitive {
			copyPrimitiveValue(src.Elem(), newPtr.Elem())
		} else {
			if err := c.copyValue(src.Elem(), newPtr.Elem()); err != nil {
				return err
			}
		}

		dst.Set(newPtr)
		return nil
	}
}

// generatePrimitiveCopy generates a copy function for primitive types
func generatePrimitiveCopy(typ reflect.Type) func(*copier, reflect.Value, reflect.Value) error {
	return func(c *copier, src, dst reflect.Value) error {
		copyPrimitiveValue(src, dst)
		return nil
	}
}

// copyPrimitiveValue performs a direct copy of primitive values
func copyPrimitiveValue(src, dst reflect.Value) {
	switch src.Kind() {
	case reflect.Bool:
		dst.SetBool(src.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst.SetInt(src.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dst.SetUint(src.Uint())
	case reflect.Float32, reflect.Float64:
		dst.SetFloat(src.Float())
	case reflect.Complex64, reflect.Complex128:
		dst.SetComplex(src.Complex())
	case reflect.String:
		dst.SetString(src.String())
	}
}

// copyPrimitiveSlice performs bulk copy for primitive slice types
func copyPrimitiveSlice(src, dst reflect.Value) {
	switch src.Type().Elem().Kind() {
	case reflect.Bool:
		copyBoolSlice(src, dst)
	case reflect.Int:
		copyIntSlice(src, dst)
	case reflect.Int8:
		copyInt8Slice(src, dst)
	case reflect.Int16:
		copyInt16Slice(src, dst)
	case reflect.Int32:
		copyInt32Slice(src, dst)
	case reflect.Int64:
		copyInt64Slice(src, dst)
	case reflect.Uint:
		copyUintSlice(src, dst)
	case reflect.Uint8:
		copyUint8Slice(src, dst)
	case reflect.Uint16:
		copyUint16Slice(src, dst)
	case reflect.Uint32:
		copyUint32Slice(src, dst)
	case reflect.Uint64:
		copyUint64Slice(src, dst)
	case reflect.Float32:
		copyFloat32Slice(src, dst)
	case reflect.Float64:
		copyFloat64Slice(src, dst)
	case reflect.String:
		copyStringSlice(src, dst)
	}
}

// Specialized slice copy functions for better performance
func copyBoolSlice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]bool)
	dstSlice := make([]bool, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyIntSlice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]int)
	dstSlice := make([]int, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyInt8Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]int8)
	dstSlice := make([]int8, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyInt16Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]int16)
	dstSlice := make([]int16, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyInt32Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]int32)
	dstSlice := make([]int32, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyInt64Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]int64)
	dstSlice := make([]int64, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyUintSlice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]uint)
	dstSlice := make([]uint, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyUint8Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]uint8)
	dstSlice := make([]uint8, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyUint16Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]uint16)
	dstSlice := make([]uint16, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyUint32Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]uint32)
	dstSlice := make([]uint32, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyUint64Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]uint64)
	dstSlice := make([]uint64, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyFloat32Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]float32)
	dstSlice := make([]float32, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyFloat64Slice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]float64)
	dstSlice := make([]float64, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

func copyStringSlice(src, dst reflect.Value) {
	srcSlice := src.Interface().([]string)
	dstSlice := make([]string, len(srcSlice))
	copy(dstSlice, srcSlice)
	dst.Set(reflect.ValueOf(dstSlice))
}

// unsafeCopy performs memory-level copy using unsafe operations
// This should be used with extreme caution and only for performance-critical scenarios
func unsafeCopy(src, dst reflect.Value) {
	if src.Kind() != dst.Kind() {
		return
	}

	switch src.Kind() {
	case reflect.Slice:
		if src.Len() == 0 {
			return
		}
		srcHeader := (*reflect.SliceHeader)(unsafe.Pointer(src.UnsafeAddr()))
		dstHeader := (*reflect.SliceHeader)(unsafe.Pointer(dst.UnsafeAddr()))
		*dstHeader = *srcHeader
	case reflect.String:
		srcHeader := (*reflect.StringHeader)(unsafe.Pointer(src.UnsafeAddr()))
		dstHeader := (*reflect.StringHeader)(unsafe.Pointer(dst.UnsafeAddr()))
		*dstHeader = *srcHeader
	}
}
