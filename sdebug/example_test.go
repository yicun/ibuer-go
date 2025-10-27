package sdebug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestExample runs all examples and demonstrates the high-performance capabilities
func TestExample(t *testing.T) {
	fmt.Println("🚀 SDebug High-Performance Examples")
	fmt.Println("=====================================")

	ExampleBasicPerson()
	ExampleStudent()
	ExampleECommerceOrder()
	ExampleHighFrequencyTrading()
	ExampleConcurrentOperations()
	ExampleErrorHandling()
	ExamplePerformanceAnalysis()

	fmt.Println("\n✅ All examples completed successfully!")
	fmt.Println("\n💡 Key takeaways:")
	fmt.Println("- Zero-configuration setup")
	fmt.Println("- Nanosecond-level performance")
	fmt.Println("- Thread-safe concurrent operations")
	fmt.Println("- Smart caching for repeated exports")
	fmt.Println("- Comprehensive error handling")
	fmt.Println("- Suitable for high-frequency, low-latency applications")
}

// TestConcurrentExampleOperations tests concurrent operations with race conditions
func TestConcurrentExampleOperations(t *testing.T) {
	// Test basic concurrent operations
	debug := NewDebugInfo(true)
	var wg sync.WaitGroup
	numGoroutines := 10
	operationsPerGoroutine := 50

	start := time.Now()

	// Simulate concurrent access from multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Each goroutine performs different operations
				err := debug.Set(fmt.Sprintf("goroutine_%d", id), "operation", j)
				assert.NoError(t, err)
				err = debug.Incr("metrics", "total_operations", 1)
				assert.NoError(t, err)

				// Simulate some work
				if j%10 == 0 {
					err = debug.Store(fmt.Sprintf("goroutine_%d", id), "checkpoint", int64(j))
					assert.NoError(t, err)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Add performance metrics
	err := debug.Set("concurrent", "goroutines", numGoroutines)
	assert.NoError(t, err)
	err = debug.Set("concurrent", "operations_per_goroutine", operationsPerGoroutine)
	assert.NoError(t, err)
	err = debug.Set("concurrent", "total_operations", numGoroutines*operationsPerGoroutine)
	assert.NoError(t, err)
	err = debug.Set("concurrent", "elapsed_ms", int(elapsed.Milliseconds()))
	assert.NoError(t, err)

	// Verify final state
	data := debug.Peek()
	assert.Contains(t, data, "metrics")
	assert.Contains(t, data, "concurrent")

	// Check total operations count
	metrics := data["metrics"].(map[string]any)
	var totalOps int64
	switch v := metrics["total_operations"].(type) {
	case int64:
		totalOps = v
	case float64:
		totalOps = int64(v)
	case *int64:
		totalOps = *v
	default:
		t.Fatalf("Unexpected type for total_operations: %T", v)
	}
	assert.Equal(t, int64(numGoroutines*operationsPerGoroutine), totalOps)
}

// TestHighFrequencyTradingExample tests high-frequency trading simulation
func TestHighFrequencyTradingExample(t *testing.T) {
	trader := &HighFrequencyTrader{
		TraderID: "TRADER-TEST",
		Symbol:   "TEST",
	}
	trader.Storage = NewDebugInfo(true)

	// Simulate high-frequency market data updates
	startTime := time.Now()
	numUpdates := 100

	for i := 0; i < numUpdates; i++ {
		// Ultra-fast debug recording
		err := trader.Storage.Store("market", "price", int64(150.0+float64(i)*0.01))
		assert.NoError(t, err)
		err = trader.Storage.Incr("metrics", "updates", 1)
		assert.NoError(t, err)
		err = trader.Storage.Set("market", "timestamp", time.Now().UnixNano())
		assert.NoError(t, err)

		// Track order book changes
		if i%10 == 0 {
			orderBook := map[string]any{
				"bid":    150.0 + float64(i)*0.01,
				"ask":    150.1 + float64(i)*0.01,
				"volume": 1000 + i,
			}
			err = trader.Storage.Set("orderbook", fmt.Sprintf("update_%d", i), orderBook)
			assert.NoError(t, err)
		}
	}

	elapsed := time.Since(startTime)
	err := trader.Storage.Set("performance", "processing_time_ns", elapsed.Nanoseconds())
	assert.NoError(t, err)
	err = trader.Storage.Set("performance", "updates_per_second", 100*1e9/elapsed.Nanoseconds())
	assert.NoError(t, err)

	// Verify performance metrics
	debugMap := trader.Storage.Peek()
	assert.Contains(t, debugMap, "performance")
	assert.Contains(t, debugMap, "metrics")

	// Check updates count
	metrics := debugMap["metrics"].(map[string]any)
	var updates int64
	switch v := metrics["updates"].(type) {
	case int64:
		updates = v
	case float64:
		updates = int64(v)
	case *int64:
		updates = *v
	default:
		t.Fatalf("Unexpected type for updates: %T", v)
	}
	assert.Equal(t, int64(numUpdates), updates)
}

// TestECommerceOrderExample tests e-commerce order processing
func TestECommerceOrderExample(t *testing.T) {
	order := &ECommerceOrder{
		OrderID:    "ORD-TEST-001",
		CustomerID: "CUST-TEST",
		Items: []OrderItem{
			{ProductID: "P001", Name: "Laptop", Quantity: 1, Price: 1299.99},
			{ProductID: "P002", Name: "Mouse", Quantity: 2, Price: 29.99},
		},
		Total:     1359.97,
		Status:    "processing",
		CreatedAt: time.Now(),
	}
	order.Storage = NewDebugInfo(true)

	// Track order processing steps
	err := order.Storage.Set("source", "", "web_checkout")
	assert.NoError(t, err)
	err = order.Storage.Set("processing", "step", "payment_verification")
	assert.NoError(t, err)
	err = order.Storage.Set("processing", "timestamp", time.Now().Format(time.RFC3339))
	assert.NoError(t, err)

	// Track inventory checks
	for i, item := range order.Items {
		inventoryData := map[string]any{
			"product_id": item.ProductID,
			"available":  true,
			"reserved":   item.Quantity,
		}
		err = order.Storage.Set("inventory", fmt.Sprintf("item_%d", i), inventoryData)
		assert.NoError(t, err)
	}

	// Simulate processing with metrics
	err = order.Storage.Incr("metrics", "inventory_checks", int64(len(order.Items)))
	assert.NoError(t, err)
	err = order.Storage.Incr("metrics", "payment_attempts", 1)
	assert.NoError(t, err)

	// Simulate validation
	validationResults := map[string]any{
		"payment":   "approved",
		"inventory": "available",
		"shipping":  "eligible",
	}
	err = order.Storage.Set("validation", "", validationResults)
	assert.NoError(t, err)

	// Verify final state
	jsonData, err := json.Marshal(order)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "ORD-TEST-001")
	assert.Contains(t, string(jsonData), "inventory")
	assert.Contains(t, string(jsonData), "validation")
}

// TestStudentExample tests student debugging with inheritance
func TestStudentExample(t *testing.T) {
	student := &Student{
		Person: Person{
			ID:   "S-TEST-9876",
			Name: "Test Student",
			Age:  20,
		},
		School: "Test University",
		Grade:  85,
	}
	student.Storage = NewDebugInfo(true)

	// Debug info works on the embedded Person
	err := student.AddDebugInfo("enrollment", "active")
	assert.NoError(t, err)
	err = student.AddDebugInfo2("academic", "gpa", 3.8)
	assert.NoError(t, err)
	err = student.Storage.Incr("metrics", "courses_taken", 4)
	assert.NoError(t, err)

	// Complex data structures
	grades := map[string]any{
		"math":      92,
		"physics":   88,
		"chemistry": 85,
	}
	err = student.AddDebugInfo("grades", grades)
	assert.NoError(t, err)

	// Verify JSON output
	jsonData, err := json.Marshal(student)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "Test Student")
	assert.Contains(t, string(jsonData), "grades")
	assert.Contains(t, string(jsonData), "academic")
}

// TestPersonExample tests basic person debugging
func TestPersonExample(t *testing.T) {
	person := &Person{
		ID:   "P-TEST-12345",
		Name: "Test Person",
		Age:  28,
	}
	person.Storage = NewDebugInfo(true)

	// Add debug information with error handling
	err := person.AddDebugInfo("source", "user_input")
	assert.NoError(t, err)
	err = person.AddDebugInfo("validation", "passed")
	assert.NoError(t, err)
	err = person.AddDebugInfo2("processing", "step", "initialization")
	assert.NoError(t, err)
	err = person.AddDebugInfo2("processing", "timestamp", time.Now().Format(time.RFC3339))
	assert.NoError(t, err)

	// Use atomic counters
	err = person.Storage.Incr("metrics", "updates", 1)
	assert.NoError(t, err)
	err = person.Storage.Incr("metrics", "views", 5)
	assert.NoError(t, err)

	// Export as JSON
	jsonData, err := json.Marshal(person)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "Test Person")
	assert.Contains(t, string(jsonData), "source")
	assert.Contains(t, string(jsonData), "metrics")

	// View debug info
	debugMap, err := person.GetDebugInfoMap()
	assert.NoError(t, err)
	assert.Contains(t, debugMap, "source")
	assert.Contains(t, debugMap, "validation")
	assert.Contains(t, debugMap, "processing")
}

// TestMemoryEstimationExample tests memory estimation functionality
func TestMemoryEstimationExample(t *testing.T) {
	debug := NewDebugInfo(true)

	// Test various data types
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
		assert.Greater(t, size, int64(0), "Size should be positive for %s", tc.name)
	}

	// Test actual storage and export performance
	start := time.Now()
	for i := 0; i < 100; i++ {
		err := debug.Set("performance", fmt.Sprintf("item_%d", i), map[string]any{
			"index":     i,
			"data":      fmt.Sprintf("test_data_%d", i),
			"timestamp": time.Now().Unix(),
		})
		assert.NoError(t, err)
	}
	storeTime := time.Since(start)

	// Measure export performance
	start = time.Now()
	data := debug.ToMap()
	exportTime := time.Since(start)

	fmt.Printf("\nPerformance metrics:\n")
	fmt.Printf("- Storage time for 100 items: %v\n", storeTime)
	fmt.Printf("- Export time: %v\n", exportTime)
	fmt.Printf("- Total items: %d\n", len(data))
	fmt.Printf("- Storage rate: %.0f items/ms\n", float64(100)/float64(storeTime.Milliseconds()))
	fmt.Printf("- Export rate: %.0f items/ms\n", float64(len(data))/float64(exportTime.Milliseconds()))

	assert.Greater(t, len(data), 0, "Should have stored data")
	// Note: For small datasets, export might be slower due to overhead, but caching helps for larger datasets
}

// TestExampleOptionalDeepCopy tests the optional deep copy example
func TestExampleOptionalDeepCopy(t *testing.T) {
	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run example
	ExampleOptionalDeepCopy()

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected content
	assert.Contains(t, output, "=== Example 8: Optional Deep Copy Feature ===")
	assert.Contains(t, output, "Deep copy enabled: true")
	assert.Contains(t, output, "Deep copy disabled: false")
	assert.Contains(t, output, "Stored name: Alice (original was protected)")
	assert.Contains(t, output, "Stored price: 149.99 (modified due to no deep copy)")
	assert.Contains(t, output, "Data integrity protected")
	assert.Contains(t, output, "Maximum performance")
}
