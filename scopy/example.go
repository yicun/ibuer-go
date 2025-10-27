package scopy

import (
	"fmt"
	"log"
)

// Example demonstrates basic usage of the scopy package
func Example() {
	// Create a copier with default options
	copier := New(nil)

	// Copy a simple value
	original := 42
	copied, err := copier.Copy(original)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Original: %v, Copied: %v\n", original, copied)

	// Copy a struct
	type Person struct {
		Name string
		Age  int
	}

	person := Person{Name: "John", Age: 30}
	copiedPerson, err := copier.Copy(person)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Original: %+v, Copied: %+v\n", person, copiedPerson)
}

// Example_complexTypes demonstrates copying complex types
func Example_complexTypes() {
	copier := New(nil)

	// Copy a slice
	numbers := []int{1, 2, 3, 4, 5}
	copiedNumbers, err := copier.Copy(numbers)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Original: %v, Copied: %v\n", numbers, copiedNumbers)

	// Copy a map
	data := map[string]int{"a": 1, "b": 2, "c": 3}
	copiedData, err := copier.Copy(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Original: %v, Copied: %v\n", data, copiedData)

	// Copy a nested struct
	type Address struct {
		Street string
		City   string
	}

	type Person struct {
		Name    string
		Age     int
		Address Address
	}

	person := Person{
		Name: "John",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "New York",
		},
	}

	copiedPerson, err := copier.Copy(person)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Original: %+v, Copied: %+v\n", person, copiedPerson)
}

// Example_pointers demonstrates copying pointer types
func Example_pointers() {
	copier := New(nil)

	// Copy a pointer to int
	value := 42
	ptr := &value
	copiedPtr, err := copier.Copy(ptr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original pointer: %p, Copied pointer: %p\n", ptr, copiedPtr)
	fmt.Printf("Original value: %v, Copied value: %v\n", *ptr, *copiedPtr.(*int))

	// Copy a struct with pointers
	type Node struct {
		Value int
		Next  *Node
	}

	node1 := &Node{Value: 1}
	node2 := &Node{Value: 2}
	node1.Next = node2

	copiedNode, err := copier.Copy(node1)
	if err != nil {
		log.Fatal(err)
	}

	copied := copiedNode.(*Node)
	fmt.Printf("Original chain: %d -> %d\n", node1.Value, node1.Next.Value)
	fmt.Printf("Copied chain: %d -> %d\n", copied.Value, copied.Next.Value)
}

// Example_options demonstrates using options
func Example_options() {
	// Create copier with custom options
	opts := &Options{
		MaxDepth:       50,
		EnableCache:    true,
		SkipZeroValues: true,
	}

	copier := New(opts)

	type Config struct {
		Name     string
		Port     int
		Debug    bool
		Timeout  int
		MaxConns int
	}

	config := Config{
		Name:     "MyApp",
		Port:     8080,
		Debug:    false, // This will be skipped
		Timeout:  30,
		MaxConns: 0, // This will be skipped
	}

	var copiedConfig Config
	err := copier.CopyTo(config, &copiedConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original: %+v\n", config)
	fmt.Printf("Copied: %+v\n", copiedConfig)
}

// Example_performance demonstrates performance optimization
func Example_performance() {
	// Enable caching for better performance with repeated struct types
	opts := &Options{
		EnableCache: true,
	}

	copier := New(opts)

	type Data struct {
		ID      int
		Name    string
		Values  []float64
		Tags    map[string]string
	}

	// Create a large dataset
	data := make([]Data, 1000)
	for i := range data {
		data[i] = Data{
			ID:     i,
			Name:   fmt.Sprintf("Item_%d", i),
			Values: []float64{1.1, 2.2, 3.3, 4.4, 5.5},
			Tags: map[string]string{
				"type":    "test",
				"version": "1.0",
			},
		}
	}

	// Copy the entire dataset
	copiedData, err := copier.Copy(data)
	if err != nil {
		log.Fatal(err)
	}

	copied := copiedData.([]Data)
	fmt.Printf("Copied %d items\n", len(copied))
	fmt.Printf("First item: %+v\n", copied[0])
}

// Example_customTypes demonstrates copying custom types
func Example_customTypes() {
	// Define a custom type
	type UserID int64
	type User struct {
		ID    UserID
		Name  string
		Email string
	}

	copier := New(nil)

	user := User{
		ID:    UserID(12345),
		Name:  "John Doe",
		Email: "john@example.com",
	}

	copiedUser, err := copier.Copy(user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original: %+v\n", user)
	fmt.Printf("Copied: %+v\n", copiedUser)
}

// Example_interfaces demonstrates copying interface types
func Example_interfaces() {
	copier := New(nil)

	// Interface with different concrete types
	var data interface{} = map[string]interface{}{
		"name": "John",
		"age":  30,
		"scores": []int{90, 85, 95},
	}

	copiedData, err := copier.Copy(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original: %v\n", data)
	fmt.Printf("Copied: %v\n", copiedData)

	// Verify deep copy
	originalMap := data.(map[string]interface{})

	// Modify original
	originalMap["name"] = "Jane"
	originalScores := originalMap["scores"].([]int)
	originalScores[0] = 100

	fmt.Printf("After modification - Original: %v\n", data)
	fmt.Printf("After modification - Copied: %v\n", copiedData)
}

// Example_errorHandling demonstrates error handling
func Example_errorHandling() {
	copier := New(nil)

	// This will work fine
	data := map[string]int{"a": 1, "b": 2}
	copied, err := copier.Copy(data)
	if err != nil {
		log.Printf("Error copying data: %v", err)
	} else {
		fmt.Printf("Successfully copied: %v\n", copied)
	}

	// Using CopyTo with validation
	type Config struct {
		Host string
		Port int
	}

	src := Config{Host: "localhost", Port: 8080}
	var dst Config

	err = copier.CopyTo(src, &dst)
	if err != nil {
		log.Printf("Error copying to destination: %v", err)
	} else {
		fmt.Printf("Successfully copied to destination: %+v\n", dst)
	}
}