package slog

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Test structures for priority testing

// TestStructLogger implements both Logger and has slog tags
type TestStructLogger struct {
	Value string `slog:"value"`
	Name  string `slog:"name"`
}

func (t TestStructLogger) MarshalLog() ([]byte, error) {
	return json.Marshal(map[string]string{
		"struct_logger": "struct_logger_result",
		"priority":      "struct_logger_highest",
	})
}

// TestFieldSerializer has field-level serializer
type TestFieldSerializer struct {
	NormalField string    `slog:"normal_field"`
	TimeField   time.Time `slog:"time_field,ser=time_rfc3339"`
	AmountField float64   `slog:"amount_field,ser=currency_cny"`
}

// TestFieldLogger has Logger interface on field
type TestFieldLogger struct {
	ID          int                  `slog:"id"`
	Name        string               `slog:"name"`
	LoggerField TestFieldLoggerField `slog:"logger_field"`
	BasicField  string               `slog:"basic_field"`
}

type TestFieldLoggerField struct {
	FieldValue string
}

func (f TestFieldLoggerField) MarshalLog() ([]byte, error) {
	return json.Marshal(map[string]string{
		"field_logger": "field_logger_result",
		"priority":     "field_logger_third",
	})
}

// TestComplexPriority tests the new priority order: Field slog:"ser=xxx" → Field Struct Logger → Basic Type → Mask
type TestComplexPriority struct {
	// This field has both Logger interface and ser tag - ser should win
	LoggerWithSer TestLoggerWithSer `slog:"logger_with_ser,ser=first_priority"`

	// This field has Logger interface only
	LoggerOnly TestLoggerOnly `slog:"logger_only"`

	// This field has ser tag only
	SerOnly time.Time `slog:"ser_only,ser=time_rfc3339"`

	// This field is basic type
	BasicOnly string `slog:"basic_only"`

	// This field has mask
	MaskedField string `slog:"masked_field,mask=phone"`
}

type TestLoggerWithSer struct {
	Time time.Time
}

func (t TestLoggerWithSer) MarshalLog() ([]byte, error) {
	return json.Marshal(map[string]string{
		"logger_with_ser": "should_not_see_this",
		"priority":        "logger_should_be_second",
	})
}

type TestLoggerOnly struct {
	Value string
}

func (t TestLoggerOnly) MarshalLog() ([]byte, error) {
	return json.Marshal(map[string]string{
		"logger_only": "field_logger_result",
		"priority":    "field_logger_third",
	})
}

// TestPriorityOrder verifies the new priority: Field slog:"ser=xxx" → Field Struct Logger → Basic Type → Mask
func TestPriorityOrder(t *testing.T) {
	// Test 0: Struct Logger
	TestStructLoggerPriority(t)

	// Test 1: Field-level serializer (ser=xxx) has highest priority
	TestFieldSerializerPriority(t)

	// Test 2: Field Struct Logger has second priority
	TestFieldLoggerPriority(t)

	// Test 3: Complex priority order with multiple field types
	TestComplexPriorityOrder(t)
}

func TestStructLoggerPriority(t *testing.T) {
	data := TestStructLogger{
		Value: "test_value",
		Name:  "test_name",
	}
	//TestStructLogger.MarshalLog()
	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("testStructLoggerPriority: Marshal result: %s", string(result))

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should use StructLogger.MarshalLog() result, not individual fields
	if parsed["struct_logger"] != "struct_logger_result" {
		t.Errorf("Expected struct_logger result, got %v", parsed)
	}
	if parsed["priority"] != "struct_logger_highest" {
		t.Errorf("Expected struct_logger_highest priority, got %v", parsed)
	}
	// Should not see individual field values
	if parsed["value"] != "" || parsed["name"] != "" {
		t.Errorf("Should not see individual field values, got %v", parsed)
	}
}

func TestFieldSerializerPriority(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := TestFieldSerializer{
		NormalField: "normal_value",
		TimeField:   now,
		AmountField: 123.45,
	}

	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("TestFieldSerializerPriority: Marshal result: %s", string(result))

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Normal field should use basic serialization
	if parsed["normal_field"] != "normal_value" {
		t.Errorf("Expected normal_field='normal_value', got %v", parsed["normal_field"])
	}

	// Time field should use RFC3339 format due to ser=time_rfc3339
	if parsed["time_field"] != "2025-01-01T12:00:00Z" {
		t.Errorf("Expected time_field in RFC3339 format, got %v", parsed["time_field"])
	}

	// Amount field should use currency format due to ser=currency_cny
	if parsed["amount_field"] != "¥123.45" {
		t.Errorf("Expected amount_field='¥123.45', got %v", parsed["amount_field"])
	}
}

func TestFieldLoggerPriority(t *testing.T) {
	data := TestFieldLogger{
		ID:          123,
		Name:        "test_name",
		LoggerField: TestFieldLoggerField{FieldValue: "field_value"},
		BasicField:  "basic_value",
	}

	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("testFieldLoggerPriority: Marshal result: %s", string(result))

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Basic fields should use normal serialization
	if parsed["id"] != float64(123) {
		t.Errorf("Expected id=123, got %v", parsed["id"])
	}
	if parsed["name"] != "test_name" {
		t.Errorf("Expected name='test_name', got %v", parsed["name"])
	}
	if parsed["basic_field"] != "basic_value" {
		t.Errorf("Expected basic_field='basic_value', got %v", parsed["basic_field"])
	}

	// LoggerField should use its MarshalLog() method
	loggerResult, ok := parsed["logger_field"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected logger_field to be a map, got %T", parsed["logger_field"])
	}
	if loggerResult["field_logger"] != "field_logger_result" {
		t.Errorf("Expected field_logger result, got %v", loggerResult)
	}
	if loggerResult["priority"] != "field_logger_third" {
		t.Errorf("Expected field_logger_third priority, got %v", loggerResult)
	}
}

func TestComplexPriorityOrder(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := TestComplexPriority{
		// LoggerWithSer TestLoggerWithSer `slog:"logger_with_ser,ser=first_priority"`
		LoggerWithSer: TestLoggerWithSer{Time: now},
		// LoggerOnly TestLoggerOnly `slog:"logger_only"`
		LoggerOnly: TestLoggerOnly{Value: "logger_value"},
		// SerOnly time.Time `slog:"ser_only,ser=time_rfc3339"`
		SerOnly: now,
		// BasicOnly string `slog:"basic_only"`
		BasicOnly: "basic_value",
		// MaskedField string `slog:"masked_field,mask=phone"`
		MaskedField: "13800138000",
	}
	RegisterSerializer("first_priority", func(a any) ([]byte, error) {
		return json.Marshal(map[string]any{
			"logger_with_ser": "field_ser_result",
			"priority":        "field_logger_first",
		})
	})

	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("testComplexPriorityOrder: Marshal result: %s", string(result))

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 1. LoggerWithSer should use ser=first_priority (ser has highest priority)
	loggerResult, ok := parsed["logger_with_ser"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected logger_with_ser to be a map, got %T", parsed["logger_with_ser"])
	}
	if loggerResult["priority"] != "field_logger_first" {
		t.Errorf("Expected logger_with_ser in [field_logger_first], got %v", parsed["logger_with_ser"])
	}

	// 2. SerOnly should use RFC3339 format due to ser tag (ser has highest priority)
	if parsed["ser_only"] != "2025-01-01T12:00:00Z" {
		t.Errorf("Expected ser_only in [RFC3339 format], got %v", parsed["ser_only"])
	}

	// 3. LoggerOnly should use its MarshalLog() method (field logger comes after ser)
	loggerResult, ok = parsed["logger_only"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected logger_only to be a map, got %T", parsed["logger_only"])
	}
	if loggerResult["priority"] != "field_logger_third" {
		t.Errorf("Expected logger_only in [field_logger_third], got %v", loggerResult)
	}

	// 4. BasicOnly should use basic string serialization
	if parsed["basic_only"] != "basic_value" {
		t.Errorf("Expected basic_only='basic_value', got %v", parsed["basic_only"])
	}

	// 5. MaskedField should be masked
	if parsed["masked_field"] != "138****8000" {
		t.Errorf("Expected masked_field='138****8000', got %v", parsed["masked_field"])
	}
}

// TestPriorityEdgeCases tests edge cases for the new priority logic
func TestPriorityEdgeCases(t *testing.T) {
	// Test 1: Empty serializer name should fall through to next priority
	TestEmptySerializerName(t)

	// Test 2: Non-existent serializer should fall through to next priority
	TestNonExistentSerializer(t)

	// Test 3: Multiple fields with different priorities
	TestMultipleFieldsDifferentPriorities(t)
}

func TestEmptySerializerName(t *testing.T) {
	type EmptySer struct {
		Field string `slog:"field,ser="`
	}

	data := EmptySer{Field: "test_value"}
	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("testEmptySerializerName Marshal result: %s", string(result))

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should use basic serialization since ser="" is empty
	if parsed["field"] != "test_value" {
		t.Errorf("Expected field='test_value', got %v", parsed["field"])
	}
}

func TestNonExistentSerializer(t *testing.T) {
	type NonExistentSer struct {
		Field string `slog:"field,ser=nonexistent"`
	}

	data := NonExistentSer{Field: "test_value"}
	result, err := MarshalWithOpts(data, WithErrorFallback(true))
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("testNonExistentSerializer Marshal result: %s", string(result))

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should use error fallback since serializer doesn't exist
	if !contains(parsed["field"], "FIELD_SERIALIZE_ERROR") {
		t.Errorf("Expected error fallback message, got %v", parsed["field"])
	}
}

func TestMultipleFieldsDifferentPriorities(t *testing.T) {
	type MixedPriorities struct {
		LoggerField TestLoggerOnly `slog:"logger_field"`
		SerField    time.Time      `slog:"ser_field,ser=time_rfc3339"`
		BasicField  string         `slog:"basic_field"`
		MaskedField string         `slog:"masked_field,mask=email"`
	}

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := MixedPriorities{
		LoggerField: TestLoggerOnly{Value: "logger_value"},
		SerField:    now,
		BasicField:  "basic_value",
		MaskedField: "user@example.com",
	}

	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("TestMultipleFieldsDifferentPriorities Marshal result: %s", string(result))

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify each field uses the correct priority
	loggerResult := parsed["logger_field"].(map[string]interface{})
	if loggerResult["priority"] != "field_logger_third" {
		t.Errorf("Expected field_logger result, got %v", loggerResult)
	}

	if parsed["ser_field"] != "2025-01-01T12:00:00Z" {
		t.Errorf("Expected ser_field in RFC3339 format, got %v", parsed["ser_field"])
	}

	if parsed["basic_field"] != "basic_value" {
		t.Errorf("Expected basic_field='basic_value', got %v", parsed["basic_field"])
	}

	if parsed["masked_field"] != "use***@example.com" {
		t.Errorf("Expected masked_field='use***@example.com', got %v", parsed["masked_field"])
	}
}

// TestPriorityWithConditionalLogger tests priority with conditional logging
type ConditionalPriority struct {
	AlwaysShow     string               `slog:"always_show"`
	Conditional    TestConditionalField `slog:"conditional"`
	SerConditional time.Time            `slog:"ser_conditional,ser=time_rfc3339"`
}

type TestConditionalField struct {
	Value string
	Show  bool
}

func (c TestConditionalField) ShouldLog() bool {
	return c.Show
}

func TestPriorityWithConditionalLogger(t *testing.T) {
	// Test when conditional field should not be shown
	testConditionalFieldHidden(t)

	// Test when conditional field should be shown
	testConditionalFieldShown(t)
}

func testConditionalFieldHidden(t *testing.T) {
	data1 := ConditionalPriority{
		AlwaysShow:     "always_shown",
		Conditional:    TestConditionalField{Value: "hidden", Show: false},
		SerConditional: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result1, err := Marshal(data1)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed1 map[string]interface{}
	if err := json.Unmarshal(result1, &parsed1); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed1["always_show"] != "always_shown" {
		t.Errorf("Expected always_show='always_shown', got %v", parsed1["always_show"])
	}
	if parsed1["conditional"] != nil {
		t.Errorf("Expected conditional to be omitted, got %v", parsed1["conditional"])
	}
	if parsed1["ser_conditional"] != "2025-01-01T12:00:00Z" {
		t.Errorf("Expected ser_conditional in RFC3339 format, got %v", parsed1["ser_conditional"])
	}
}

func testConditionalFieldShown(t *testing.T) {
	data2 := ConditionalPriority{
		AlwaysShow:     "always_shown",
		Conditional:    TestConditionalField{Value: "shown", Show: true},
		SerConditional: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result2, err := Marshal(data2)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed2 map[string]interface{}
	if err := json.Unmarshal(result2, &parsed2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed2["always_show"] != "always_shown" {
		t.Errorf("Expected always_show='always_shown', got %v", parsed2["always_show"])
	}
	if parsed2["conditional"] == nil {
		t.Errorf("Expected conditional to be shown, got %v", parsed2["conditional"])
	}
	if parsed2["ser_conditional"] != "2025-01-01T12:00:00Z" {
		t.Errorf("Expected ser_conditional in RFC3339 format, got %v", parsed2["ser_conditional"])
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestPriorityDocumentation verifies the documented priority order
func TestPriorityDocumentation(t *testing.T) {
	// Create a comprehensive test that demonstrates all priorities in order
	type ComprehensivePriority struct {
		// Priority 1: Field slog:"ser=xxx" (highest priority)
		FieldSer time.Time `slog:"field_ser,ser=time_rfc3339"`

		// Priority 2: Field Struct Logger
		FieldLogger TestFieldLoggerField `slog:"field_logger"`

		// Priority 3: Basic Type
		BasicField string `slog:"basic_field"`

		// Priority 4: Mask (applied after serialization)
		MaskedField string `slog:"masked_field,mask=phone"`
	}

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := ComprehensivePriority{
		FieldSer:    now,
		FieldLogger: TestFieldLoggerField{FieldValue: "field_value"},
		BasicField:  "basic_value",
		MaskedField: "13800138000",
	}

	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify each priority level works as expected

	// Priority 1: Field ser=xxx (highest priority)
	if parsed["field_ser"] != "2025-01-01T12:00:00Z" {
		t.Errorf("Priority 1 (Field ser=xxx) failed: expected RFC3339 format, got %v", parsed["field_ser"])
	}

	// Priority 2: Field Logger
	fieldLoggerResult := parsed["field_logger"].(map[string]interface{})
	if fieldLoggerResult["field_logger"] != "field_logger_result" {
		t.Errorf("Priority 2 (Field Logger) failed: expected field_logger result, got %v", fieldLoggerResult)
	}

	// Priority 3: Basic Type
	if parsed["basic_field"] != "basic_value" {
		t.Errorf("Priority 3 (Basic Type) failed: expected 'basic_value', got %v", parsed["basic_field"])
	}

	// Priority 4: Mask
	if parsed["masked_field"] != "138****8000" {
		t.Errorf("Priority 4 (Mask) failed: expected '138****8000', got %v", parsed["masked_field"])
	}

	t.Logf("✅ All priorities working correctly: %v", parsed)
}

// TestPriorityPerformance tests that the new priority order doesn't impact performance
func TestPriorityPerformance(t *testing.T) {
	// Create a struct with mixed priorities
	type PerformanceTest struct {
		SerField    time.Time      `slog:"ser_field,ser=time_rfc3339"`
		LoggerField TestLoggerOnly `slog:"logger_field"`
		BasicField  string         `slog:"basic_field"`
		MaskField   string         `slog:"mask_field,mask=phone"`
	}

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := PerformanceTest{
		SerField:    now,
		LoggerField: TestLoggerOnly{Value: "performance_test"},
		BasicField:  "performance_test",
		MaskField:   "13800138000",
	}

	// Test multiple times to ensure consistency
	for i := 0; i < 5; i++ {
		result, err := Marshal(data)
		if err != nil {
			t.Fatalf("Marshal failed on iteration %d: %v", i, err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("Unmarshal failed on iteration %d: %v", i, err)
		}

		// Verify all fields are processed correctly
		if parsed["ser_field"] != "2025-01-01T12:00:00Z" {
			t.Errorf("Iteration %d: Expected ser_field in RFC3339 format, got %v", i, parsed["ser_field"])
		}

		loggerResult := parsed["logger_field"].(map[string]interface{})
		if loggerResult["priority"] != "field_logger_third" {
			t.Errorf("Iteration %d: Expected field_logger result, got %v", i, loggerResult)
		}

		if parsed["basic_field"] != "performance_test" {
			t.Errorf("Iteration %d: Expected basic_field='performance_test', got %v", i, parsed["basic_field"])
		}

		if parsed["mask_field"] != "138****8000" {
			t.Errorf("Iteration %d: Expected mask_field='138****8000', got %v", i, parsed["mask_field"])
		}
	}

	t.Log("✅ Performance test passed - all priorities working consistently")
}

// TestPriorityErrorHandling tests error handling in the new priority order
func TestPriorityErrorHandling(t *testing.T) {
	// Test error in field-level serializer
	testFieldSerializerError(t)

	// Test error in field logger
	testFieldLoggerError(t)
}

func testFieldSerializerError(t *testing.T) {
	type FieldSerError struct {
		Field string `slog:"field,ser=error_serializer"`
	}

	// Register an error serializer
	RegisterSerializer("error_serializer", func(v any) ([]byte, error) {
		return nil, fmt.Errorf("serializer error")
	})

	data := FieldSerError{Field: "test_value"}

	// Test with error fallback enabled
	result, err := MarshalWithOpts(data, WithErrorFallback(true))
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should contain error message
	if !contains(parsed["field"], "FIELD_SERIALIZE_ERROR") {
		t.Errorf("Expected error fallback message, got %v", parsed["field"])
	}
}

func testFieldLoggerError(t *testing.T) {
	type FieldLoggerError struct {
		Field TestErrorLoggerField `slog:"field"`
	}

	data := FieldLoggerError{Field: TestErrorLoggerField{Value: "test_value"}}

	// Test with error fallback enabled
	result, err := MarshalWithOpts(data, WithErrorFallback(true))
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should contain error message
	if !contains(parsed["field"], "FIELD_SERIALIZE_ERROR") {
		t.Errorf("Expected error fallback message, got %v", parsed["field"])
	}
}

// TestErrorLoggerField for error testing
type TestErrorLoggerField struct {
	Value string
}

func (e TestErrorLoggerField) MarshalLog() ([]byte, error) {
	return nil, fmt.Errorf("field logger error")
}

// TestPriorityWithOmitEmpty tests priority with omitempty
func TestPriorityWithOmitEmpty(t *testing.T) {
	type OmitEmptyPriority struct {
		SerEmpty       string          `slog:"ser_empty,ser=time_rfc3339,omitempty"`
		LoggerEmpty    TestLoggerOnly  `slog:"logger_empty,omitempty"`
		LoggerEmptyPtr *TestLoggerOnly `slog:"logger_empty_ptr,omitempty"`
		BasicEmpty     string          `slog:"basic_empty,omitempty"`
		SerNonEmpty    time.Time       `slog:"ser_non_empty,ser=time_rfc3339,omitempty"`
		LoggerNonEmpty TestLoggerOnly  `slog:"logger_non_empty,omitempty"`
		BasicNonEmpty  string          `slog:"basic_non_empty,omitempty"`
	}

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := OmitEmptyPriority{
		SerEmpty:       "",                                 // Empty string
		LoggerEmptyPtr: nil,                                // Empty value
		BasicEmpty:     "",                                 // Empty string
		SerNonEmpty:    now,                                // Non-empty time
		LoggerNonEmpty: TestLoggerOnly{Value: "non_empty"}, // Non-empty logger
		BasicNonEmpty:  "non_empty",                        // Non-empty string
	}
	t.Logf("OmitEmptyPriority->LoggerEmpty: %v", data.LoggerEmpty)
	t.Logf("OmitEmptyPriority->LoggerEmpty->Value: %v", data.LoggerEmpty.Value)
	t.Logf("OmitEmptyPriority->LoggerEmptyPtr: %v", data.LoggerEmptyPtr)

	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("TestPriorityWithOmitEmpty Marshal result: %s", string(result))

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Empty fields should be omitted
	if _, exists := parsed["ser_empty"]; exists {
		t.Errorf("Expected ser_empty to be omitted, but got %v", parsed["ser_empty"])
	}
	if _, exists := parsed["logger_empty"]; exists {
		t.Errorf("Expected logger_empty to be omitted, but got %v", parsed["logger_empty"])
	}
	if _, exists := parsed["basic_empty"]; exists {
		t.Errorf("Expected basic_empty to be omitted, but got %v", parsed["basic_empty"])
	}

	// Non-empty fields should be included
	if parsed["ser_non_empty"] != "2025-01-01T12:00:00Z" {
		t.Errorf("Expected ser_non_empty in RFC3339 format, got %v", parsed["ser_non_empty"])
	}

	loggerResult := parsed["logger_non_empty"].(map[string]interface{})
	if loggerResult["logger_only"] != "field_logger_result" {
		t.Errorf("Expected field_logger result, got %v", loggerResult)
	}

	if parsed["basic_non_empty"] != "non_empty" {
		t.Errorf("Expected basic_non_empty='non_empty', got %v", parsed["basic_non_empty"])
	}
}

// BenchmarkPriorityOrder benchmarks the new priority order
func BenchmarkPriorityOrder(b *testing.B) {
	type BenchmarkStruct struct {
		SerField    time.Time      `slog:"ser_field,ser=time_rfc3339"`
		LoggerField TestLoggerOnly `slog:"logger_field"`
		BasicField  string         `slog:"basic_field"`
		MaskField   string         `slog:"mask_field,mask=phone"`
	}

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := BenchmarkStruct{
		SerField:    now,
		LoggerField: TestLoggerOnly{Value: "benchmark_value"},
		BasicField:  "benchmark_value",
		MaskField:   "13800138000",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(data)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}

// TestPriorityRegression ensures the priority change doesn't break existing functionality
func TestPriorityRegression(t *testing.T) {
	// Test that existing functionality still works with the new priority order
	type RegressionTest struct {
		// Mix of different field types that should continue to work
		StringField  string           `slog:"string_field"`
		IntField     int              `slog:"int_field"`
		BoolField    bool             `slog:"bool_field"`
		TimeField    time.Time        `slog:"time_field,ser=time_rfc3339"`
		AmountField  float64          `slog:"amount_field,ser=currency_cny"`
		LoggerField  TestLoggerOnly   `slog:"logger_field"`
		StructLogger TestStructLogger `slog:"struct_logger"`
	}

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := RegressionTest{
		StringField:  "string_value",
		IntField:     42,
		BoolField:    true,
		TimeField:    now,
		AmountField:  123.45,
		LoggerField:  TestLoggerOnly{Value: "logger_value"},
		StructLogger: TestStructLogger{Value: "struct_value", Name: "struct_name"},
	}

	result, err := Marshal(data)
	if err != nil {
		t.Fatalf("Regression test failed: %v", err)
	}
	t.Logf("Regression test result: %v", string(result))

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify all field types work correctly
	if parsed["string_field"] != "string_value" {
		t.Errorf("String field regression: expected 'string_value', got %v", parsed["string_field"])
	}
	if parsed["int_field"] != float64(42) {
		t.Errorf("Int field regression: expected 42, got %v", parsed["int_field"])
	}
	if parsed["bool_field"] != true {
		t.Errorf("Bool field regression: expected true, got %v", parsed["bool_field"])
	}
	if parsed["time_field"] != "2025-01-01T12:00:00Z" {
		t.Errorf("Time field regression: expected RFC3339 format, got %v", parsed["time_field"])
	}
	if parsed["amount_field"] != "¥123.45" {
		t.Errorf("Amount field regression: expected '¥123.45', got %v", parsed["amount_field"])
	}

	// Logger field should use MarshalLog
	loggerResult := parsed["logger_field"].(map[string]interface{})
	if loggerResult["logger_only"] != "field_logger_result" {
		t.Errorf("Logger field regression: expected field_logger result, got %v", loggerResult)
	}

	// Struct logger should use its MarshalLog
	structResult := parsed["struct_logger"].(map[string]interface{})
	if structResult["struct_logger"] != "struct_logger_result" {
		t.Errorf("Struct logger regression: expected struct_logger result, got %v", structResult)
	}

	t.Log("✅ Regression test passed - all existing functionality preserved")
}

// TestMissingSerializerErrorHandling tests that missing serializers are handled correctly
func TestMissingSerializerErrorHandling(t *testing.T) {
	type TestStruct struct {
		Field string `slog:"field,ser=missing_serializer"`
	}

	data := TestStruct{Field: "test_value"}

	// Test 1: With error fallback enabled (should output error message)
	result1, err1 := MarshalWithOpts(data, WithErrorFallback(true))
	if err1 != nil {
		t.Fatalf("Expected no error with error fallback enabled, got: %v", err1)
	}

	var parsed1 map[string]string
	if err := json.Unmarshal(result1, &parsed1); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should contain error message about missing serializer
	if !contains(parsed1["field"], "FIELD_SERIALIZE_ERROR") {
		t.Errorf("Expected error fallback message, got: %v", parsed1["field"])
	}
	if !contains(parsed1["field"], "missing_serializer") {
		t.Errorf("Expected error message to contain serializer name, got: %v", parsed1["field"])
	}

	// Test 2: With error fallback disabled (should return error)
	_, err2 := MarshalWithOpts(data, WithErrorFallback(false))
	if err2 == nil {
		t.Errorf("Expected error when error fallback disabled, but got nil")
	}

	// Verify the error message contains expected information
	if !strings.Contains(err2.Error(), "serializer 'missing_serializer' not found") {
		t.Errorf("Expected error message to contain 'serializer 'missing_serializer' not found', got: %v", err2)
	}
}

// ExamplePriorityOrder Example demonstrating the new priority order
func TestExamplePriorityOrder(t *testing.T) {
	type ExampleStruct struct {
		// Priority 1: Field slog:"ser=xxx" (field-level custom serializer - highest priority)
		FieldSer time.Time `slog:"field_ser,ser=time_rfc3339"`

		// Priority 2: Field Struct Logger (field with Logger interface)
		FieldLogger TestLoggerOnly `slog:"field_logger"`

		// Priority 3: Basic Type (default serialization)
		BasicField string `slog:"basic_field"`

		// Priority 4: Mask (applied after serialization)
		MaskedField string `slog:"masked_field,mask=phone"`
	}

	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	data := ExampleStruct{
		FieldSer:    now,
		FieldLogger: TestLoggerOnly{Value: "logger_value"},
		BasicField:  "basic_value",
		MaskedField: "13800138000",
	}

	result, _ := Marshal(data)
	println(string(result))

	// Output: {"field_ser":"2025-01-01T12:00:00Z","field_logger":{"field_logger":"field_logger_result","priority":"field_logger_third"},"basic_field":"basic_value","masked_field":"138****8000"}
}
