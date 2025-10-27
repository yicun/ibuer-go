package scopy

import (
	"testing"
)

// Benchmark basic types
func BenchmarkCopyInt(b *testing.B) {
	c := New(nil)
	src := 42

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopyString(b *testing.B) {
	c := New(nil)
	src := "hello world"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark complex types
func BenchmarkCopySlice(b *testing.B) {
	c := New(nil)
	src := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopyMap(b *testing.B) {
	c := New(nil)
	src := map[string]int{
		"a": 1, "b": 2, "c": 3, "d": 4, "e": 5,
		"f": 6, "g": 7, "h": 8, "i": 9, "j": 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark structs
func BenchmarkCopySimpleStruct(b *testing.B) {
	type SimpleStruct struct {
		Name  string
		Age   int
		Score float64
	}

	c := New(nil)
	src := SimpleStruct{
		Name:  "John Doe",
		Age:   30,
		Score: 95.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopyComplexStruct(b *testing.B) {
	type Address struct {
		Street string
		City   string
		Zip    string
	}

	type Person struct {
		Name     string
		Age      int
		Email    string
		Address  Address
		Tags     []string
		Metadata map[string]interface{}
		IsActive bool
		Score    float64
	}

	c := New(nil)
	src := Person{
		Name:  "John Doe",
		Age:   30,
		Email: "john@example.com",
		Address: Address{
			Street: "123 Main St",
			City:   "New York",
			Zip:    "10001",
		},
		Tags:     []string{"developer", "gopher", "backend"},
		Metadata: map[string]interface{}{"level": "senior", "years": 5},
		IsActive: true,
		Score:    95.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark with caching
func BenchmarkCopyStructWithCache(b *testing.B) {
	type TestStruct struct {
		ID    int
		Name  string
		Value float64
	}

	opts := &Options{
		EnableCache: true,
		MaxDepth:    100,
	}
	c := New(opts)

	src := TestStruct{
		ID:    123,
		Name:  "Test Item",
		Value: 42.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark pointers
func BenchmarkCopyPointer(b *testing.B) {
	c := New(nil)
	src := 42
	ptr := &src

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(ptr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark nested structures
func BenchmarkCopyNestedStruct(b *testing.B) {
	type Node struct {
		Value int
		Next  *Node
	}

	c := New(nil)

	// Create a linked list: 1 -> 2 -> 3 -> 4 -> 5
	node5 := &Node{Value: 5}
	node4 := &Node{Value: 4, Next: node5}
	node3 := &Node{Value: 3, Next: node4}
	node2 := &Node{Value: 2, Next: node3}
	node1 := &Node{Value: 1, Next: node2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(node1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark large data structures
func BenchmarkCopyLargeSlice(b *testing.B) {
	c := New(nil)

	// Create a large slice
	src := make([]int, 1000)
	for i := range src {
		src[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopyLargeMap(b *testing.B) {
	c := New(nil)

	// Create a large map
	src := make(map[string]int)
	for i := 0; i < 100; i++ {
		src[string(rune('a'+i%26))+string(rune('a'+i/26))] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Copy(src)
		if err != nil {
			b.Fatal(err)
		}
	}
}
