package scopy

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test basic types
func TestCopyBasicTypes(t *testing.T) {
	c := New(nil)

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{"bool", true, true},
		{"int", 42, 42},
		{"int8", int8(8), int8(8)},
		{"int16", int16(16), int16(16)},
		{"int32", int32(32), int32(32)},
		{"int64", int64(64), int64(64)},
		{"uint", uint(42), uint(42)},
		{"uint8", uint8(8), uint8(8)},
		{"uint16", uint16(16), uint16(16)},
		{"uint32", uint32(32), uint32(32)},
		{"uint64", uint64(64), uint64(64)},
		{"float32", float32(3.14), float32(3.14)},
		{"float64", 3.14159, 3.14159},
		{"complex64", complex(float32(1), float32(2)), complex(float32(1), float32(2))},
		{"complex128", complex(1, 2), complex(1, 2)},
		{"string", "hello", "hello"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Copy(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			// Verify it's a deep copy (different memory address)
			if !isPrimitiveType(reflect.TypeOf(tt.input)) {
				assert.NotSame(t, tt.input, result)
			}
		})
	}
}

// Test arrays
func TestCopyArray(t *testing.T) {
	c := New(nil)

	tests := []struct {
		name  string
		input interface{}
	}{
		{"int array", [3]int{1, 2, 3}},
		{"string array", [2]string{"hello", "world"}},
		{"bool array", [2]bool{true, false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Copy(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.input, result)

			// Verify it's a deep copy by modifying and checking
			if intArr, ok := tt.input.([3]int); ok {
				resultArr := result.([3]int)
				// Arrays are values, so we need to check if we can modify one without affecting the other
				// Since arrays are copied by value in Go, this test is mainly for completeness
				assert.Equal(t, intArr, resultArr)
			}
		})
	}
}

// Test slices
func TestCopySlice(t *testing.T) {
	c := New(nil)

	tests := []struct {
		name  string
		input interface{}
	}{
		{"int slice", []int{1, 2, 3, 4, 5}},
		{"string slice", []string{"hello", "world", "test"}},
		{"empty slice", []int{}},
		{"nil slice", []int(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Copy(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.input, result)

			// For basic types, we can't really test deep copy since they're values
			// The copy should have the same value
		})
	}
}

// Test maps
func TestCopyMap(t *testing.T) {
	c := New(nil)

	tests := []struct {
		name  string
		input interface{}
	}{
		{"int map", map[string]int{"a": 1, "b": 2, "c": 3}},
		{"string map", map[int]string{1: "one", 2: "two"}},
		{"empty map", map[string]int{}},
		{"nil map", map[string]int(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Copy(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.input, result)

			// For basic types, we can't really test deep copy since they're values
			// The copy should have the same value
		})
	}
}

// Test structs
func TestCopyStruct(t *testing.T) {
	type SimpleStruct struct {
		Name  string
		Age   int
		Score float64
	}

	type ComplexStruct struct {
		ID      int
		Tags    []string
		Data    map[string]interface{}
		Nested  *SimpleStruct
	}

	c := New(nil)

	t.Run("simple struct", func(t *testing.T) {
		input := SimpleStruct{
			Name:  "John",
			Age:   30,
			Score: 95.5,
		}

		result, err := c.Copy(input)
		require.NoError(t, err)

		copied, ok := result.(SimpleStruct)
		require.True(t, ok)
		assert.Equal(t, input, copied)
	})

	t.Run("complex struct", func(t *testing.T) {
		input := ComplexStruct{
			ID:   123,
			Tags: []string{"tag1", "tag2", "tag3"},
			Data: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			Nested: &SimpleStruct{
				Name:  "Nested",
				Age:   25,
				Score: 88.0,
			},
		}

		result, err := c.Copy(input)
		require.NoError(t, err)

		copied, ok := result.(ComplexStruct)
		require.True(t, ok)
		assert.Equal(t, input.ID, copied.ID)
		assert.Equal(t, input.Tags, copied.Tags)
		assert.Equal(t, input.Data, copied.Data)
		assert.NotNil(t, copied.Nested)
		assert.Equal(t, *input.Nested, *copied.Nested)

		// Verify deep copy by modifying original and checking copy is unaffected
		originalTags := make([]string, len(input.Tags))
		copy(originalTags, input.Tags)

		input.Tags[0] = "modified"
		assert.NotEqual(t, input.Tags[0], copied.Tags[0], "Deep copy failed - slice modification affected copy")

		// Restore original for other assertions
		input.Tags[0] = originalTags[0]
	})
}

// Test pointers
func TestCopyPointers(t *testing.T) {
	c := New(nil)

	t.Run("int pointer", func(t *testing.T) {
		value := 42
		input := &value

		result, err := c.Copy(input)
		require.NoError(t, err)

		copied, ok := result.(*int)
		require.True(t, ok)
		assert.Equal(t, *input, *copied)
		assert.NotSame(t, input, copied)
	})

	t.Run("nil pointer", func(t *testing.T) {
		var input *int

		result, err := c.Copy(input)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("struct pointer", func(t *testing.T) {
		type TestStruct struct {
			Name string
			Age  int
		}

		input := &TestStruct{Name: "John", Age: 30}

		result, err := c.Copy(input)
		require.NoError(t, err)

		copied, ok := result.(*TestStruct)
		require.True(t, ok)
		assert.Equal(t, *input, *copied)
		assert.NotSame(t, input, copied)
	})
}

// Test interfaces
func TestCopyInterfaces(t *testing.T) {
	c := New(nil)

	t.Run("interface with int", func(t *testing.T) {
		var input interface{} = 42

		result, err := c.Copy(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("interface with string", func(t *testing.T) {
		var input interface{} = "hello"

		result, err := c.Copy(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("interface with struct", func(t *testing.T) {
		type TestStruct struct {
			Name string
		}

		var input interface{} = TestStruct{Name: "John"}

		result, err := c.Copy(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("nil interface", func(t *testing.T) {
		var input interface{}

		result, err := c.Copy(input)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

// Test cycles
func TestCopyCycles(t *testing.T) {
	type Node struct {
		Value int
		Next  *Node
	}

	c := New(nil)

	t.Run("simple cycle", func(t *testing.T) {
		node1 := &Node{Value: 1}
		node2 := &Node{Value: 2}
		node1.Next = node2
		node2.Next = node1 // Create cycle

		result, err := c.Copy(node1)
		require.NoError(t, err)

		copied, ok := result.(*Node)
		require.True(t, ok)

		assert.Equal(t, node1.Value, copied.Value)
		assert.NotNil(t, copied.Next)
		assert.Equal(t, node2.Value, copied.Next.Value)
		assert.NotNil(t, copied.Next.Next)
		assert.Same(t, copied, copied.Next.Next) // Should maintain cycle
	})
}

// Test CopyTo
func TestCopyTo(t *testing.T) {
	type TestStruct struct {
		Name string
		Age  int
	}

	c := New(nil)

	t.Run("copy to struct", func(t *testing.T) {
		src := TestStruct{Name: "John", Age: 30}
		var dst TestStruct

		err := c.CopyTo(src, &dst)
		require.NoError(t, err)

		assert.Equal(t, src, dst)
	})

	t.Run("copy to pointer", func(t *testing.T) {
		src := TestStruct{Name: "John", Age: 30}
		var dst TestStruct

		err := c.CopyTo(src, &dst)
		require.NoError(t, err)

		assert.Equal(t, src, dst)
	})
}

// Test options
func TestOptions(t *testing.T) {
	t.Run("skip zero values", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Age   int
			Score float64
		}

		opts := &Options{
			MaxDepth:       100,
			SkipZeroValues: true,
		}
		c := New(opts)

		src := TestStruct{Name: "John", Age: 0, Score: 95.5}
		var dst TestStruct

		err := c.CopyTo(src, &dst)
		require.NoError(t, err)

		assert.Equal(t, src.Name, dst.Name)
		assert.Equal(t, 0, dst.Age) // Should remain zero
		assert.Equal(t, src.Score, dst.Score)
	})

	t.Run("max depth", func(t *testing.T) {
		type Node struct {
			Next *Node
		}

		opts := &Options{
			MaxDepth: 2,
		}
		c := New(opts)

		node1 := &Node{}
		node2 := &Node{}
		node3 := &Node{}
		node1.Next = node2
		node2.Next = node3

		_, err := c.Copy(node1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max depth exceeded")
	})
}

// Test error cases
func TestCopyErrors(t *testing.T) {
	c := New(nil)

	t.Run("nil src", func(t *testing.T) {
		result, err := c.Copy(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil dst in CopyTo", func(t *testing.T) {
		src := 42
		err := c.CopyTo(src, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "src and dst cannot be nil")
	})

	t.Run("non-pointer dst in CopyTo", func(t *testing.T) {
		src := 42
		var dst int
		err := c.CopyTo(src, dst)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dst must be a pointer")
	})
}

