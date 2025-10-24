package sdebug

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Person demonstrates basic SDebugInfo embedding for a person entity
type Person struct {
	SDebugInfo

	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// Student extends Person with school information
type Student struct {
	Person
	School string `json:"school"`
	Grade  int    `json:"grade"`
}

// ECommerceOrder demonstrates complex debugging scenarios
type ECommerceOrder struct {
	SDebugInfo

	OrderID    string      `json:"order_id"`
	CustomerID string      `json:"customer_id"`
	Items      []OrderItem `json:"items"`
	Total      float64     `json:"total"`
	Status     string      `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// HighFrequencyTrader demonstrates ultra-high-performance debugging
type HighFrequencyTrader struct {
	SDebugInfo

	TraderID string `json:"trader_id"`
	Symbol   string `json:"symbol"`
}

// ExampleBasicPerson demonstrates basic person debugging with error handling
func ExampleBasicPerson() {
	fmt.Println("=== Example 1: Basic Person Debugging ===")

	person := &Person{
		ID:   "12345",
		Name: "Alice Johnson",
		Age:  28,
	}

	// Add debug information with error handling
	if err := person.AddDebugInfo("source", "user_input"); err != nil {
		log.Printf("AddDebugInfo error: %v", err)
	}
	if err := person.AddDebugInfo("validation", "passed"); err != nil {
		log.Printf("AddDebugInfo error: %v", err)
	}
	if err := person.AddDebugInfo2("processing", "step", "initialization"); err != nil {
		log.Printf("AddDebugInfo2 error: %v", err)
	}
	if err := person.AddDebugInfo2("processing", "timestamp", time.Now().Format(time.RFC3339)); err != nil {
		log.Printf("AddDebugInfo2 error: %v", err)
	}

	// Use atomic counters (direct storage access for performance)
	if person.Storage != nil {
		if err := person.Storage.Incr("metrics", "updates", 1); err != nil {
			log.Printf("Incr error: %v", err)
		}
		if err := person.Storage.Incr("metrics", "views", 5); err != nil {
			log.Printf("Incr error: %v", err)
		}
	}

	// Export as JSON
	jsonData, err := json.Marshal(person)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}

	fmt.Printf("Person with debug data: %s\n", jsonData)

	// View debug info
	debugMap, err := person.GetDebugInfoMap()
	if err != nil {
		log.Printf("GetDebugInfoMap error: %v", err)
	} else {
		fmt.Printf("Debug info: %+v\n\n", debugMap)
	}
}

// ExampleStudent demonstrates student debugging with inheritance
func ExampleStudent() {
	fmt.Println("=== Example 2: Student with Inherited Debugging ===")

	student := &Student{
		Person: Person{
			ID:   "S9876",
			Name: "Bob Smith",
			Age:  20,
		},
		School: "MIT",
		Grade:  85,
	}

	// Debug info works on the embedded Person
	if err := student.AddDebugInfo("enrollment", "active"); err != nil {
		log.Printf("AddDebugInfo error: %v", err)
	}
	if err := student.AddDebugInfo2("academic", "gpa", 3.8); err != nil {
		log.Printf("AddDebugInfo2 error: %v", err)
	}
	if student.Storage != nil {
		if err := student.Storage.Incr("metrics", "courses_taken", 4); err != nil {
			log.Printf("Incr error: %v", err)
		}
	}

	// Complex data structures
	grades := map[string]any{
		"math":      92,
		"physics":   88,
		"chemistry": 85,
	}
	if err := student.AddDebugInfo("grades", grades); err != nil {
		log.Printf("AddDebugInfo error: %v", err)
	}

	jsonData, err := json.Marshal(student)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	fmt.Printf("Student with debug data: %s\n\n", jsonData)
}

// ExampleECommerceOrder demonstrates e-commerce order processing
func ExampleECommerceOrder() {
	fmt.Println("=== Example 3: E-Commerce Order Processing ===")

	order := &ECommerceOrder{
		OrderID:    "ORD-2024-001",
		CustomerID: "CUST-789",
		Items: []OrderItem{
			{ProductID: "P001", Name: "Laptop", Quantity: 1, Price: 1299.99},
			{ProductID: "P002", Name: "Mouse", Quantity: 2, Price: 29.99},
		},
		Total:     1359.97,
		Status:    "processing",
		CreatedAt: time.Now(),
	}

	// Track order processing steps
	if err := order.AddDebugInfo("source", "web_checkout"); err != nil {
		log.Printf("AddDebugInfo error: %v", err)
	}
	if err := order.AddDebugInfo2("processing", "step", "payment_verification"); err != nil {
		log.Printf("AddDebugInfo2 error: %v", err)
	}
	if err := order.AddDebugInfo2("processing", "timestamp", time.Now().Format(time.RFC3339)); err != nil {
		log.Printf("AddDebugInfo2 error: %v", err)
	}

	// Track inventory checks
	for i, item := range order.Items {
		inventoryData := map[string]any{
			"product_id": item.ProductID,
			"available":  true,
			"reserved":   item.Quantity,
		}
		if err := order.AddDebugInfo2("inventory", fmt.Sprintf("item_%d", i), inventoryData); err != nil {
			log.Printf("AddDebugInfo2 error: %v", err)
		}
	}

	// Simulate processing with metrics
	if order.Storage != nil {
		if err := order.Storage.Incr("metrics", "inventory_checks", int64(len(order.Items))); err != nil {
			log.Printf("Incr error: %v", err)
		}
		if err := order.Storage.Incr("metrics", "payment_attempts", 1); err != nil {
			log.Printf("Incr error: %v", err)
		}
	}

	// Simulate validation
	validationResults := map[string]any{
		"payment":   "approved",
		"inventory": "available",
		"shipping":  "eligible",
	}
	if err := order.AddDebugInfo("validation", validationResults); err != nil {
		log.Printf("AddDebugInfo error: %v", err)
	}

	jsonData, err := json.Marshal(order)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	fmt.Printf("Order with debug data: %s\n\n", jsonData)
}

// ExampleHighFrequencyTrading demonstrates high-frequency trading simulation
func ExampleHighFrequencyTrading() {
	fmt.Println("=== Example 4: High-Frequency Trading Simulation ===")

	trader := &HighFrequencyTrader{
		TraderID: "TRADER-001",
		Symbol:   "AAPL",
	}

	// Simulate high-frequency market data updates
	startTime := time.Now()

	// Simulate 1000 market updates with nanosecond-level debugging
	for i := 0; i < 1000; i++ {
		// Ultra-fast debug recording (direct storage access for maximum performance)
		if trader.Storage != nil {
			if err := trader.Storage.Store("market", "price", int64(150.0+float64(i)*0.01)); err != nil {
				log.Printf("Store error: %v", err)
			}
			if err := trader.Storage.Incr("metrics", "updates", 1); err != nil {
				log.Printf("Incr error: %v", err)
			}
			if err := trader.Storage.Set("market", "timestamp", time.Now().UnixNano()); err != nil {
				log.Printf("Set error: %v", err)
			}
		}

		// Track order book changes
		if i%10 == 0 {
			orderBook := map[string]any{
				"bid":    150.0 + float64(i)*0.01,
				"ask":    150.1 + float64(i)*0.01,
				"volume": 1000 + i,
			}
			if err := trader.AddDebugInfo("orderbook", orderBook); err != nil {
				log.Printf("AddDebugInfo error: %v", err)
			}
		}
	}

	elapsed := time.Since(startTime)
	if err := trader.AddDebugInfo2("performance", "processing_time_ns", elapsed.Nanoseconds()); err != nil {
		log.Printf("AddDebugInfo2 error: %v", err)
	}
	if err := trader.AddDebugInfo2("performance", "updates_per_second", 1000*1e9/elapsed.Nanoseconds()); err != nil {
		log.Printf("AddDebugInfo2 error: %v", err)
	}

	debugMap, err := trader.GetDebugInfoMap()
	if err != nil {
		log.Printf("GetDebugInfoMap error: %v", err)
	} else {
		fmt.Printf("Trading debug summary:\n")
		if perf, ok := debugMap["performance"].(map[string]any); ok {
			fmt.Printf("- Processing time: %v ns\n", perf["processing_time_ns"])
			fmt.Printf("- Updates per second: %v\n", perf["updates_per_second"])
		}
		if metrics, ok := debugMap["metrics"].(map[string]any); ok {
			fmt.Printf("- Total updates: %v\n\n", metrics["updates"])
		}
	}
}

// ExampleConcurrentOperations demonstrates concurrent operations with race conditions
func ExampleConcurrentOperations() {
	fmt.Println("=== Example 5: Concurrent Operations with Race Conditions ===")

	debug := NewDebugInfo(true)
	var wg sync.WaitGroup
	numGoroutines := 50
	operationsPerGoroutine := 100

	start := time.Now()

	// Simulate concurrent access from multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Each goroutine performs different operations
				if err := debug.Set(fmt.Sprintf("goroutine_%d", id), "operation", j); err != nil {
					log.Printf("Set error: %v", err)
				}
				if err := debug.Incr("metrics", "total_operations", 1); err != nil {
					log.Printf("Incr error: %v", err)
				}

				// Simulate some work
				if j%10 == 0 {
					if err := debug.Store(fmt.Sprintf("goroutine_%d", id), "checkpoint", int64(j)); err != nil {
						log.Printf("Store error: %v", err)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Add performance metrics using direct storage access
	if err := debug.Set("concurrent", "goroutines", numGoroutines); err != nil {
		log.Printf("Set error: %v", err)
	}
	if err := debug.Set("concurrent", "operations_per_goroutine", operationsPerGoroutine); err != nil {
		log.Printf("Set error: %v", err)
	}
	if err := debug.Set("concurrent", "total_operations", numGoroutines*operationsPerGoroutine); err != nil {
		log.Printf("Set error: %v", err)
	}
	if err := debug.Set("concurrent", "elapsed_ms", int(elapsed.Milliseconds())); err != nil {
		log.Printf("Set error: %v", err)
	}

	// Export results
	jsonData, err := debug.ToJSON()
	if err != nil {
		log.Printf("ToJSON error: %v", err)
	} else {
		fmt.Printf("Concurrent operations completed in %d ms\n", elapsed.Milliseconds())
		fmt.Printf("Performance: %d ops/ms\n", (numGoroutines*operationsPerGoroutine)/int(elapsed.Milliseconds()))
		fmt.Printf("Sample JSON output (truncated): %s...\n\n", string(jsonData)[:200])
	}
}

// ExampleErrorHandling demonstrates error handling and edge cases
func ExampleErrorHandling() {
	fmt.Println("=== Example 6: Error Handling and Edge Cases ===")

	debug := NewDebugInfo(true)

	// Test various error conditions
	testCases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "Empty topKey",
			fn: func() error {
				return debug.Set("", "subkey", "value")
			},
		},
		{
			name: "Valid operation",
			fn: func() error {
				return debug.Set("test", "key", "value")
			},
		},
		{
			name: "Complex data structure",
			fn: func() error {
				complexData := map[string]any{
					"nested": map[string]any{
						"deep": map[string]any{
							"value": "test",
						},
					},
				}
				return debug.Set("complex", "data", complexData)
			},
		},
	}

	for _, tc := range testCases {
		err := tc.fn()
		if err != nil {
			fmt.Printf("❌ %s: %v\n", tc.name, err)
		} else {
			fmt.Printf("✅ %s: success\n", tc.name)
		}
	}

	// Test disabled state
	disabledDebug := NewDebugInfo(false)
	err := disabledDebug.Set("test", "key", "value")
	fmt.Printf("Disabled debug operations: %v (should be nil)\n", err)

	data := disabledDebug.Peek()
	fmt.Printf("Data from disabled debug: %+v\n\n", data)
}

// ExamplePerformanceAnalysis demonstrates memory and performance analysis
func ExamplePerformanceAnalysis() {
	fmt.Println("=== Example 7: Memory and Performance Analysis ===")

	debug := NewDebugInfo(true)

	// Test memory estimation
	testData := []struct {
		name string
		data any
	}{
		{"String", "Hello, World!"},
		{"Integer", 42},
		{"Float", 3.14159},
		{"Boolean", true},
		{"Byte slice", []byte("test data")},
		{"String array", []any{"a", "b", "c"}},
		{"Complex map", map[string]any{
			"key1": "value1",
			"key2": 123,
			"key3": []any{"nested", "array"},
		}},
	}

	fmt.Println("Memory size estimation:")
	for _, tc := range testData {
		size := debug.estimateSize(tc.data)
		fmt.Printf("- %s: %d bytes\n", tc.name, size)
	}

	// Test actual storage and export performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		if err := debug.Set("performance", fmt.Sprintf("item_%d", i), map[string]any{
			"index":     i,
			"data":      fmt.Sprintf("test_data_%d", i),
			"timestamp": time.Now().Unix(),
		}); err != nil {
			log.Printf("Set error: %v", err)
		}
	}
	storeTime := time.Since(start)

	// Measure export performance
	start = time.Now()
	data := debug.ToMap()
	exportTime := time.Since(start)

	fmt.Printf("\nPerformance metrics:\n")
	fmt.Printf("- Storage time for 1000 items: %v\n", storeTime)
	fmt.Printf("- Export time: %v\n", exportTime)
	fmt.Printf("- Total items: %d\n", len(data))
	fmt.Printf("- Storage rate: %.0f items/ms\n", float64(1000)/float64(storeTime.Milliseconds()))
	fmt.Printf("- Export rate: %.0f items/ms\n", float64(len(data))/float64(exportTime.Milliseconds()))
}

// ExampleOptionalDeepCopy demonstrates the optional deep copy feature for performance optimization
func ExampleOptionalDeepCopy() {
	fmt.Println("=== Example 8: Optional Deep Copy Feature ===")

	// Create debug instance with deep copy enabled (default for safety)
	debug := NewDebugInfo(true)
	fmt.Printf("Deep copy enabled: %v\n", debug.IsDeepCopyEnabled())

	// Create mutable data structure
	userData := map[string]any{
		"name":  "Alice",
		"score": 100,
		"metadata": map[string]any{
			"level":    "premium",
			"lastSeen": time.Now().Format(time.RFC3339),
		},
	}

	// Store the data
	if err := debug.Set("user", "profile", userData); err != nil {
		log.Printf("Set error: %v", err)
		return
	}

	// Modify original data (this should NOT affect stored data when deep copy is enabled)
	userData["name"] = "Bob"
	userData["score"] = 50
	userData["metadata"].(map[string]any)["level"] = "basic"

	// Verify stored data is protected
	storedData := debug.Peek()
	if profile, ok := storedData["user"].(map[string]any)["profile"].(map[string]any); ok {
		fmt.Printf("Stored name: %v (original was protected)\n", profile["name"])
		fmt.Printf("Stored score: %v (original was protected)\n", profile["score"])
		if metadata, ok := profile["metadata"].(map[string]any); ok {
			fmt.Printf("Stored level: %v (original was protected)\n", metadata["level"])
		}
	}

	// Now disable deep copy for maximum performance
	debug.SetDeepCopy(false)
	fmt.Printf("\nDeep copy disabled: %v\n", debug.IsDeepCopyEnabled())

	// Create new mutable data
	productData := map[string]any{
		"id":    "P001",
		"price": 99.99,
		"specs": map[string]any{
			"color": "red",
			"size":  "large",
		},
	}

	// Store the data
	if err := debug.Set("product", "info", productData); err != nil {
		log.Printf("Set error: %v", err)
		return
	}

	// Modify original data (this WILL affect stored data when deep copy is disabled)
	productData["price"] = 149.99
	productData["specs"].(map[string]any)["color"] = "blue"

	// Verify stored data is affected (no protection)
	storedData = debug.Peek()
	if product, ok := storedData["product"].(map[string]any)["info"].(map[string]any); ok {
		fmt.Printf("Stored price: %v (modified due to no deep copy)\n", product["price"])
		if specs, ok := product["specs"].(map[string]any); ok {
			fmt.Printf("Stored color: %v (modified due to no deep copy)\n", specs["color"])
		}
	}

	fmt.Println("\n💡 Key insights:")
	fmt.Println("- Deep copy enabled: Data integrity protected, ~3.4x slower")
	fmt.Println("- Deep copy disabled: Maximum performance, external modifications affect stored data")
	fmt.Println("- Choose based on your performance vs. safety requirements")
}
