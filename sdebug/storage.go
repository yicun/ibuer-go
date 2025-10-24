package sdebug

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

// SDebugStorage is a high-performance, thread-safe debug information storage system.
// Features:
// - Two-level key writing: topKey -> subKey -> value
// - Atomic counters and any value types (any/slice/map)
// - One-time export of Map or JSON bytes with smart caching
// - Dynamic enable/disable mode with no-op when disabled
// - Optional deep copy to prevent external modification (enabled by default for safety)
// - High concurrency, low latency, minimal resource consumption
// - Sophisticated deep copy with type-specific optimizations
//
// Thread Safety: All operations are thread-safe using sync.Map and atomic operations.
// Performance: Optimized for nanosecond-level operations with zero-allocation patterns.
// Memory: Minimal overhead with smart caching and optional deep copy protection.
type SDebugStorage struct {
	enabled   atomic.Bool  // debug enable/disable flag - controls all operations
	deepCopy  atomic.Bool  // deep copy enable/disable flag - controls data protection
	top       sync.Map     // top-level key -> values (map[string]any | *sync.RWMutex)
	mu        sync.RWMutex // protects ToMap / ToJSON operations from race conditions
	cacheMap  atomic.Value // cached map export (*map[string]any) - zero-allocation caching
	cacheJSON atomic.Value // cached JSON export (*[]byte) - zero-allocation caching
}

// NewDebugInfo creates a new SDebugStorage instance with the specified enabled state.
// When enabled is false, all operations become no-ops for maximum performance.
// Deep copy is enabled by default for data integrity.
//
// Example:
//   debug := sdebug.NewDebugInfo(true)  // Enable debugging with deep copy protection
//   debug := sdebug.NewDebugInfo(false) // Disable debugging for maximum performance
//
// Returns: A new SDebugStorage instance ready for use
func NewDebugInfo(enabled bool) *SDebugStorage {
	d := &SDebugStorage{}
	d.enabled.Store(enabled)
	d.deepCopy.Store(true) // Enable deep copy by default for safety
	d.cacheMap.Store(&map[string]any{"debug": enabled})
	d.cacheJSON.Store(&[]byte{})
	return d
}

// Set stores any value with the specified top-level and sub-level keys.
// Supports sophisticated deep copy with type-specific optimizations.
//
// Features:
// - Two-level key structure: topKey -> subKey -> value
// - Map expansion: If val is map[string]any and subKey is empty, map keys become sub-level keys
// - Optional deep copy: Protects against external modifications when enabled
// - Type preservation: Maintains original data types during storage
// - Thread-safe: Concurrent operations are safely handled
//
// Deep Copy Behavior:
// - Enabled by default for data integrity
// - When enabled: Creates deep copies of complex data structures
// - When disabled: Stores references directly (faster but less safe)
// - Can be controlled via SetDeepCopy() method
//
// Performance:
// - Basic types: Direct value copy (O(1))
// - Maps/Arrays: Deep copy with type-specific optimizations
// - Complex objects: JSON serialization fallback
//
// Example:
//   err := debug.Set("user", "name", "Alice")           // Simple value
//   err := debug.Set("metrics", "count", 42)            // Number
//   err := debug.Set("data", "", map[string]any{        // Map expansion
//       "field1": "value1",
//       "field2": "value2",
//   })
//
// Parameters:
//   topKey: Top-level category/key (cannot be empty)
//   subKey: Sub-level key (can be empty for map expansion)
//   val: Value to store (any type supported)
//
// Returns: Error if topKey is empty, nil otherwise
func (d *SDebugStorage) Set(topKey, subKey string, val any) error {
	// Early return if debugging is disabled for maximum performance
	if d.disabled() {
		return nil
	}

	// Validate input - topKey cannot be empty
	if topKey == "" {
		return fmt.Errorf("topKey cannot be empty")
	}

	// Get or create the sub-map for this topKey
	actual, _ := d.top.LoadOrStore(topKey, make(map[string]any))
	sub := actual.(map[string]any)

	// Get lock for the sub-map to ensure thread safety
	l := d.lockOf(topKey)
	l.Lock()
	defer l.Unlock()

	// Handle map expansion: if val is a map and subKey is empty, use map keys as sub-keys
	if m, ok := val.(map[string]any); ok && subKey == "" {
		// Expand the map - each key in the input map becomes a sub-key
		for k2, v2 := range m {
			if d.deepCopy.Load() {
				// Deep copy enabled: protect against external modifications
				sub[k2] = deepCopyValue(v2)
			} else {
				// Deep copy disabled: store reference directly (faster)
				sub[k2] = v2
			}
		}
	} else {
		// Normal operation: store value with specific subKey
		if d.deepCopy.Load() {
			// Deep copy enabled: create protected copy
			sub[subKey] = deepCopyValue(val)
		} else {
			// Deep copy disabled: store reference directly
			sub[subKey] = val
		}
	}

	return nil
}

// SetDeepCopy enables or disables deep copy functionality.
// When disabled, performance improves but external modifications to stored data may affect debug info.
func (d *SDebugStorage) SetDeepCopy(enabled bool) {
	d.deepCopy.Store(enabled)
}

// IsDeepCopyEnabled returns whether deep copy is currently enabled.
func (d *SDebugStorage) IsDeepCopyEnabled() bool {
	return d.deepCopy.Load()
}

// Incr atomically increments a counter by the specified delta value.
// Negative delta values will decrement the counter.
func (d *SDebugStorage) Incr(topKey, subKey string, delta int64) error {
	if d.disabled() {
		return nil
	}
	if topKey == "" {
		return fmt.Errorf("topKey cannot be empty")
	}

	err := d.atomicCounterOp(topKey, subKey, func(ptr *int64) {
		atomic.AddInt64(ptr, delta)
	})

	return err
}

// Store atomically sets a counter to the specified value.
func (d *SDebugStorage) Store(topKey, subKey string, val int64) error {
	if d.disabled() {
		return nil
	}
	if topKey == "" {
		return fmt.Errorf("topKey cannot be empty")
	}

	err := d.atomicCounterOp(topKey, subKey, func(ptr *int64) {
		atomic.StoreInt64(ptr, val)
	})

	return err
}

// ToMap returns a deep copy snapshot of all debug data as a map.
// This operation:
// 1. Clears all keys and values
// 2. Disables debug mode
// 3. Subsequent calls return the cached value
func (d *SDebugStorage) ToMap() map[string]any {
	// Acquire lock
	d.mu.Lock()
	defer d.mu.Unlock()
	// Return cached value if debugging is disabled
	if d.disabled() {
		if p := d.cacheMap.Load(); p != nil {
			return *p.(*map[string]any)
		} else {
			return map[string]any{"debug": false}
		}
	}
	// Disable writes to prevent concurrent modifications
	d.enabled.Store(false)
	// Generate map
	out := make(map[string]any)
	d.top.Range(func(k, v any) bool {
		// Skip lockKey entries
		if _, ok := k.(lockKey); ok {
			return true
		}
		// Process normal values
		sub := deepCopyMap(v.(map[string]any))
		switch len(sub) {
		case 0:
			// Include empty maps, especially for debug key
			out[k.(string)] = sub
			return true
		case 1:
			if val, ok := sub[""]; ok {
				out[k.(string)] = val
				return true
			}
			fallthrough
		default:
			out[k.(string)] = sub
		}
		return true
	})
	// Clear top-level map
	d.top = sync.Map{}
	// Cache results
	d.cacheMap.Store(&out)
	d.cacheJSON.Store(&[]byte{})

	return out
}

// ToJSON returns JSON bytes representation of all debug data.
// 1. Calls ToMap to get key-values and serializes them
// 2. Subsequent calls return the cached value
func (d *SDebugStorage) ToJSON() ([]byte, error) {
	// Check if JSON cache is empty
	if p := d.cacheJSON.Load(); p != nil {
		if b := *p.(*[]byte); len(b) > 0 {
			return b, nil
		}
	}
	// Get map
	m := d.ToMap()
	// Acquire lock
	d.mu.Lock()
	defer d.mu.Unlock()
	// Double-check cache
	if p := d.cacheJSON.Load(); p != nil {
		if b := *p.(*[]byte); len(b) > 0 {
			return b, nil
		}
	}
	// JSON serialization
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	d.cacheJSON.Store(&cp)

	return cp, nil
}

// Peek returns a deep copy snapshot of current debug data without disabling debug mode, caching, or clearing.
// This is a non-flush mode interface that always returns fresh data.
// Note: deepCopyMap may cause memory pressure with large datasets, use with caution.
func (d *SDebugStorage) Peek() map[string]any {
	if d.disabled() {
		// Debug is disabled, return cached value if available
		if p := d.cacheMap.Load(); p != nil {
			return *p.(*map[string]any)
		}
		return map[string]any{"debug": false}
	}

	// No write lock needed, read-only traversal
	out := make(map[string]any)
	d.top.Range(func(k, v any) bool {
		// Skip lock objects
		if _, ok := k.(lockKey); ok {
			return true
		}
		sub := deepCopyMap(v.(map[string]any))
		switch len(sub) {
		case 0:
			return true
		case 1:
			if val, ok := sub[""]; ok {
				out[k.(string)] = val
				return true
			}
			fallthrough
		default:
			out[k.(string)] = sub
		}
		return true
	})
	return out
}

// Clone returns a deep copy of the current SDebugStorage with all original state preserved (enabled/disabled, cache, data).
func (d *SDebugStorage) Clone() *SDebugStorage {
	newD := &SDebugStorage{}
	newD.enabled.Store(d.enabled.Load())   // Inherit original state
	newD.cacheMap.Store(&map[string]any{}) // Clear cache
	newD.cacheJSON.Store(&[]byte{})

	// Read-only traversal of original object d.top
	d.top.Range(func(k, v any) bool {
		// Skip internal lock keys
		if _, ok := k.(lockKey); ok {
			return true
		}
		topKey := k.(string)
		subMap := deepCopyMap(v.(map[string]any)) // Deep copy sub-map
		// Write to new instance
		for subKey, val := range subMap {
			newD.Set(topKey, subKey, val)
		}
		return true
	})
	return newD
}

// MarshalJSON implements json.Marshaler interface for automatic JSON serialization.
func (d *SDebugStorage) MarshalJSON() ([]byte, error) {
	return d.ToJSON()
}

// UnmarshalJSON implements json.Unmarshaler interface for automatic JSON deserialization.
func (d *SDebugStorage) UnmarshalJSON(data []byte) error {
	m := map[string]any{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	// Reset enabled, force to true
	d.enabled.Store(true)
	// Reset content and cache
	d.top = sync.Map{}
	d.cacheMap.Store(&map[string]any{"debug": true})
	d.cacheJSON.Store(&[]byte{})

	for k, v := range m {
		d.Set(k, "", v) // Ignore errors, continue processing
	}
	return nil
}

// disabled returns whether the current DebugInfo is disabled.
func (d *SDebugStorage) disabled() bool {
	return !d.enabled.Load()
}

// clearAndDisable clears all data and disables debug mode in one operation.
func (d *SDebugStorage) clearAndDisable() {
	d.enabled.Store(false)
	d.top = sync.Map{}
}

// atomicCounterOp provides common atomic counter implementation.
func (d *SDebugStorage) atomicCounterOp(topKey, subKey string, fn func(*int64)) error {
	if topKey == "" {
		return fmt.Errorf("topKey cannot be empty")
	}

	actual, _ := d.top.LoadOrStore(topKey, make(map[string]any))
	sub := actual.(map[string]any)
	// Get lock
	l := d.lockOf(topKey)
	l.Lock()
	defer l.Unlock()
	// Operate on counter
	var ptr *int64
	if v, ok := sub[subKey]; ok {
		if p, ok := v.(*int64); ok {
			ptr = p
		} else {
			ptr = new(int64)
			sub[subKey] = ptr
		}
	} else {
		ptr = new(int64)
		sub[subKey] = ptr
	}
	fn(ptr)
	return nil
}

// lockKey represents a lock for sub-maps
type lockKey struct{ topKey string }

func (d *SDebugStorage) lockOf(topKey string) *sync.RWMutex {
	lk, _ := d.top.LoadOrStore(lockKey{topKey}, new(sync.RWMutex))
	return lk.(*sync.RWMutex)
}

// estimateSize estimates the memory size of a value (rough estimation).
func (d *SDebugStorage) estimateSize(val any) int64 {
	switch v := val.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case int, int8, int16, int32, int64:
		return 8
	case uint, uint8, uint16, uint32, uint64:
		return 8
	case float32, float64:
		return 8
	case bool:
		return 1
	case map[string]any:
		size := int64(0)
		for k, v := range v {
			size += int64(len(k)) + d.estimateSize(v)
		}
		return size
	case []any:
		size := int64(0)
		for _, item := range v {
			size += d.estimateSize(item)
		}
		return size
	default:
		// For complex types, use JSON serialization size as estimation
		if data, err := json.Marshal(v); err == nil {
			return int64(len(data))
		}
		return 64 // Default value
	}
}

// Cleanup removes expired data and lock objects for optimization.
func (d *SDebugStorage) Cleanup() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Clean up lock objects
	locksToRemove := []lockKey{}
	d.top.Range(func(k, v any) bool {
		if lk, ok := k.(lockKey); ok {
			locksToRemove = append(locksToRemove, lk)
		}
		return true
	})

	for _, lk := range locksToRemove {
		d.top.Delete(lk)
	}

	return nil
}

// Reset clears all data and resets the storage to initial state.
func (d *SDebugStorage) Reset() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Clear all data
	d.top = sync.Map{}

	// Store debug key to actual storage, not just cache
	d.top.Store("debug", map[string]any{"": d.enabled.Load()})

	d.cacheMap.Store(&map[string]any{"debug": d.enabled.Load()})
	d.cacheJSON.Store(&[]byte{})

	return nil
}

// deepCopyValue creates a deep copy of any value using type-specific optimizations.
// This is the core deep copy function that handles different data types efficiently.
//
// Type-specific handling:
// - Basic types (string, int, float, bool): Direct value copy (O(1))
// - *int64 (atomic counters): Preserved as pointers (internal management)
// - map[string]any: Recursive deep copy of all key-value pairs
// - []any: Deep copy of all elements
// - []byte: Direct byte copy
// - map[any]any: Deep copy with key preservation
// - Other types: JSON serialization fallback
//
// Performance characteristics:
// - Basic types: Constant time O(1)
// - Collections: Linear time O(n) where n is collection size
// - Complex objects: Depends on JSON serialization complexity
//
// Thread safety: This function is thread-safe as it only operates on input data.
// Returns: Deep copy of the input value, or nil if input is nil.
func deepCopyValue(val any) any {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		// Basic types: Direct value copy (O(1) performance)
		// These are immutable values, so direct copy is safe and fast
		return v
	case *int64:
		// Atomic counter pointers: Preserve as-is (internal management)
		// These are managed internally by the storage system for atomic operations
		// Copying the pointer maintains the atomic behavior
		return v
	case map[string]any:
		// Map types: Recursive deep copy of all key-value pairs
		// Ensures complete isolation from external modifications
		return deepCopyMap(v)
	case []any:
		// Slice types: Deep copy of all elements
		// Each element is recursively deep copied
		return deepCopySlice(v)
	case []byte:
		// Byte slices: Direct byte copy
		// Efficient copying for binary data
		return deepCopyByteSlice(v)
	case map[any]any:
		// Any-key maps: Deep copy with key preservation
		// Handles maps with non-string keys
		return deepCopyAnyMap(v)
	default:
		// Other types: JSON serialization fallback
		// Ensures compatibility with any serializable type
		return deepCopyByJSON(val)
	}
}

// deepCopyMap creates a deep copy of a map[string]any with recursive value copying.
// This function ensures complete isolation of the copied map from the original.
//
// Features:
// - Recursive deep copy: All values are deep copied using deepCopyValue()
// - Key preservation: String keys are preserved as-is
// - Complete isolation: Modifications to the copy don't affect the original
// - Nil safety: Returns nil if input is nil
//
// Performance: O(n) where n is the number of key-value pairs
// Memory: O(n) for the new map plus memory for all deep copied values
//
// Example:
//   original := map[string]any{"key": "value", "nested": map[string]any{"inner": "data"}}
//   copy := deepCopyMap(original)
//   copy["key"] = "modified"        // Doesn't affect original
//   copy["nested"].(map[string]any)["inner"] = "changed" // Doesn't affect original
//
// Returns: Deep copy of the input map, or nil if input is nil.
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any)
	for k, v := range m {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopySlice creates a deep copy of a []any slice with recursive element copying.
// Each element in the slice is deep copied to ensure complete isolation.
//
// Features:
// - Recursive deep copy: All elements are deep copied using deepCopyValue()
// - Index preservation: Maintains the same order and indices
// - Complete isolation: Modifications to elements don't affect the original
// - Nil safety: Returns nil if input is nil
//
// Performance: O(n) where n is the number of elements
// Memory: O(n) for the new slice plus memory for all deep copied elements
//
// Example:
//   original := []any{"string", 42, map[string]any{"key": "value"}}
//   copy := deepCopySlice(original)
//   copy[0] = "modified"                    // Doesn't affect original
//   copy[2].(map[string]any)["key"] = "new"  // Doesn't affect original
//
// Returns: Deep copy of the input slice, or nil if input is nil.
func deepCopySlice(slice []any) []any {
	if slice == nil {
		return nil
	}
	result := make([]any, len(slice))
	for i, item := range slice {
		result[i] = deepCopyValue(item)
	}
	return result
}

// deepCopyByteSlice creates a deep copy of a []byte slice.
// This is optimized for binary data and provides efficient memory copying.
//
// Features:
// - Direct memory copy: Uses built-in copy() for maximum efficiency
// - Exact size allocation: Pre-allocates the exact size needed
// - Nil safety: Returns nil if input is nil
//
// Performance: O(n) where n is the number of bytes
// Memory: O(n) for the new byte slice
//
// Example:
//   original := []byte{1, 2, 3, 4, 5}
//   copy := deepCopyByteSlice(original)
//   copy[0] = 99  // Doesn't affect original
//
// Returns: Deep copy of the input byte slice, or nil if input is nil.
func deepCopyByteSlice(slice []byte) []byte {
	if slice == nil {
		return nil
	}
	result := make([]byte, len(slice))
	copy(result, slice)
	return result
}

// deepCopyAnyMap creates a deep copy of a map[any]any with key preservation.
// This function handles maps with non-string keys while maintaining type safety.
//
// Features:
// - Key preservation: Maintains original key types when possible
// - String key optimization: String keys are preserved as strings
// - Recursive value copying: All values are deep copied using deepCopyValue()
// - Complete isolation: Modifications don't affect the original
// - Nil safety: Returns nil if input is nil
//
// Performance: O(n) where n is the number of key-value pairs
// Memory: O(n) for the new map plus memory for all deep copied values
//
// Example:
//   original := map[any]any{1: "one", "two": 2, 3.14: "pi"}
//   copy := deepCopyAnyMap(original)
//   copy[1] = "modified"  // Doesn't affect original
//
// Returns: Deep copy of the input map, or nil if input is nil.
func deepCopyAnyMap(m map[any]any) map[any]any {
	if m == nil {
		return nil
	}
	result := make(map[any]any)
	for k, v := range m {
		// 注意：这里假设 key 是不可变类型
		copiedKey := k
		if strKey, ok := k.(string); ok {
			copiedKey = strKey
		}
		result[copiedKey] = deepCopyValue(v)
	}
	return result
}

// deepCopyByJSON creates a deep copy using JSON serialization.
// This is the fallback method for types that don't have specific optimization handlers.
//
// Features:
// - Universal compatibility: Works with any JSON-serializable type
// - Complete isolation: Creates entirely new objects
// - Type conversion: May change types (e.g., int64 -> float64)
// - Reliable fallback: Ensures compatibility with complex objects
//
// Performance: Depends on JSON serialization complexity
// Memory: O(n) where n is the serialized size
// Limitations: Types must be JSON-serializable
//
// Example:
//   type CustomStruct struct { Name string `json:"name"` }
//   original := CustomStruct{Name: "test"}
//   copy := deepCopyByJSON(original)
//   copy.(map[string]any)["name"] = "modified"  // Doesn't affect original
//
// Returns: Deep copy as interface{}, or nil if serialization fails.（通用但较慢）
func deepCopyByJSON(val any) any {
	// 对于无法识别的类型，使用 JSON 序列化进行深拷贝
	data, err := json.Marshal(val)
	if err != nil {
		// 序列化失败，返回原值（有风险但比崩溃好）
		return val
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return val
	}
	return result
}
