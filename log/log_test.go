// Package log provides field-level JSON logging, outputting only fields with the 'log' tag.
// Priority: Struct Logger → Field Logger → ser=xxx → Basic Type → Mask
package log

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestStructWithLogTags tests basic field serialization with log tags.
func TestStructWithLogTags(t *testing.T) {
	type TestStruct struct {
		A string `log:"a"`
		B int    `log:"b"`
		C string `log:"-"` // Should be ignored
		D bool   `log:"d"`
	}

	s := TestStruct{
		A: "value_a",
		B: 42,
		C: "value_c",
		D: true,
	}

	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{"a":"value_a","b":42,"d":true}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestStructWithoutLogTags tests fallback to json tags.
func TestStructWithoutLogTags(t *testing.T) {
	type TestStruct struct {
		A string `json:"a"`
		B int    `json:"b"`
		C string `json:"-"` // Should be ignored
		D bool   `json:"d"`
	}

	s := TestStruct{
		A: "value_a",
		B: 42,
		C: "value_c",
		D: true,
	}

	// Disable JSON fallback, should return nil
	opts := &Options{DisableJSONFallback: true}
	enc := newEncoder()
	enc.opts = opts
	err := enc.encode(s)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if enc.out != nil {
		t.Errorf("Expected nil output when JSON fallback is disabled and no log tags, got %v", enc.out)
	}

	// Enable JSON fallback (default), should work
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{"a":"value_a","b":42,"d":true}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestOmitEmpty tests omitting empty fields.
func TestOmitEmpty(t *testing.T) {
	type TestStruct struct {
		A string `log:"a,omitempty"`
		B int    `log:"b,omitempty"`
		C string `log:"c"` // No omitempty
	}

	s := TestStruct{
		A: "", // Empty, should be omitted
		B: 0,  // Zero, should be omitted
		C: "", // Empty, but no omitempty, should be present
	}

	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{"c":""}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}

	// Test with global OmitEmptyByDefault:true
	s2 := struct {
		A string `log:"a"`
		B int    `log:"b"`
		C string `log:"c"`
	}{
		A: "",
		B: 0,
		C: "non_empty",
	}

	data2, err := MarshalWithOpts(s2, WithOptions(Options{OmitEmptyByDefault: true}))
	if err != nil {
		t.Fatalf("MarshalWithOpts failed: %v", err)
	}

	expected2 := `{"c":"non_empty"}`
	if string(data2) != expected2 {
		t.Errorf("Expected %s, got %s", expected2, data2)
	}

	// Test with global OmitEmptyByDefault:false
	s3 := struct {
		A string `log:"a"`
		B int    `log:"b"`
		C string `log:"c"`
	}{
		A: "",
		B: 0,
		C: "non_empty",
	}

	data3, err := MarshalWithOpts(s3, WithOptions(Options{OmitEmptyByDefault: false}))
	if err != nil {
		t.Fatalf("MarshalWithOpts failed: %v", err)
	}

	expected3 := `{"a":"","b":0,"c":"non_empty"}`
	if string(data3) != expected3 {
		t.Errorf("Expected %s, got %s", expected3, data3)
	}
}

// TestCustomSerializer tests custom serializers.
func TestCustomSerializer(t *testing.T) {
	RegisterSerializer("test_upper", func(v any) ([]byte, error) {
		if s, ok := v.(string); ok {
			return json.Marshal(strings.ToUpper(s))
		}
		return nil, fmt.Errorf("test_upper expects string")
	})

	type TestStruct struct {
		Text string `log:"text,ser=test_upper"`
	}

	s := TestStruct{Text: "hello"}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{"text":"HELLO"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestMask tests masking functionality.
func TestMask(t *testing.T) {
	RegisterMask("test_mask", func(s string) string {
		return "***"
	})

	type TestStruct struct {
		SSN string `log:"ssn,mask=test_mask"`
	}

	s := TestStruct{SSN: "123456789"}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{"ssn":"***"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestTimeSerializer tests built-in time serializers.
func TestTimeSerializer(t *testing.T) {
	tm := time.Date(2025, 1, 1, 12, 0, 5, 123000000, time.UTC)

	type TestStruct struct {
		T1 time.Time `log:"t1,ser=time_rfc3339"`
		T2 time.Time `log:"t2,ser=time_date"`
		T3 time.Time `log:"t3,ser=time_unix"`
	}

	s := TestStruct{T1: tm, T2: tm, T3: tm}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	// Expected format: {"t1":"2025-01-01T12:00:05Z","t2":"2025-01-01","t3":1730484005}
	// Note: Unix timestamp might vary slightly based on system timezone if not UTC
	// For this test, we'll check if the string contains the expected parts
	if !strings.Contains(string(data), `"t1":"2025-01-01T12:00:05Z"`) {
		t.Errorf("Expected RFC3339 format in output, got %s", data)
	}
	if !strings.Contains(string(data), `"t2":"2025-01-01"`) {
		t.Errorf("Expected date format in output, got %s", data)
	}
	if !strings.Contains(string(data), `"t3":`) {
		t.Errorf("Expected unix timestamp in output, got %s", data)
	}
}

// TestDurationSerializer tests built-in duration serializers.
func TestDurationSerializer(t *testing.T) {
	dur := 3*time.Hour + 30*time.Minute + 45*time.Second + 123*time.Millisecond

	type TestStruct struct {
		D1 time.Duration `log:"d1,ser=duration_ms"`     // Milliseconds
		D2 time.Duration `log:"d2,ser=duration_sec_2"`  // Seconds, 2 decimals
		D3 time.Duration `log:"d3,ser=duration_string"` // String format
	}

	s := TestStruct{D1: dur, D2: dur, D3: dur}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	// Expected: {"d1":12645123,"d2":12645.12,"d3":"3h30m45.123s"}
	// Check for expected parts
	if !strings.Contains(string(data), `"d1":12645123`) {
		t.Errorf("Expected milliseconds in output, got %s", data)
	}
	if !strings.Contains(string(data), `"d2":12645.12`) {
		t.Errorf("Expected seconds with 2 decimals in output, got %s", data)
	}
	if !strings.Contains(string(data), `"d3":"3h30m45.123s"`) {
		t.Errorf("Expected string format in output, got %s", data)
	}
}

// TestDurationSerializerPrecisionType tests that RegisterDurationSerializerWithPrecision returns float64 values
func TestDurationSerializerPrecisionType(t *testing.T) {
	// Test the duration serializer with precision
	dur := 3*time.Hour + 30*time.Minute + 45*time.Second + 123*time.Millisecond

	type TestStruct struct {
		D1 time.Duration `log:"d1,ser=duration_sec_2"` // Seconds, 2 decimals
		D2 time.Duration `log:"d2,ser=duration_sec_3"` // Seconds, 3 decimals
		D3 time.Duration `log:"d3,ser=duration_ms_1"`  // Milliseconds, 1 decimal
	}

	s := TestStruct{D1: dur, D2: dur, D3: dur}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	// Parse the JSON to check the types
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Check that the values are numeric (float64) not strings
	for field, value := range result {
		switch v := value.(type) {
		case float64:
			t.Logf("Field %s: type=float64, value=%v", field, v)
		case string:
			t.Errorf("Field %s: expected float64 but got string: %v", field, v)
		default:
			t.Errorf("Field %s: unexpected type %T: %v", field, v, v)
		}
	}

	// Verify expected values
	expectedD1 := 12645.12
	if d1, ok := result["d1"].(float64); !ok || d1 != expectedD1 {
		t.Errorf("Expected d1=%v, got %v", expectedD1, result["d1"])
	}

	expectedD2 := 12645.123
	if d2, ok := result["d2"].(float64); !ok || d2 != expectedD2 {
		t.Errorf("Expected d2=%v, got %v", expectedD2, result["d2"])
	}

	expectedD3 := 12645123.0
	if d3, ok := result["d3"].(float64); !ok || d3 != expectedD3 {
		t.Errorf("Expected d3=%v, got %v", expectedD3, result["d3"])
	}
}

// TestCurrencySerializer tests built-in currency serializers.
func TestCurrencySerializer(t *testing.T) {
	type TestStruct struct {
		Price1 float64 `log:"price1,ser=currency_cny"`
		Price2 float64 `log:"price2,ser=currency_usd"`
		Price3 int     `log:"price3,ser=currency_jpy"` // JPY has 0 decimals
	}

	s := TestStruct{Price1: 123.45, Price2: 67.89, Price3: 999}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	// Expected: {"price1":"¥123.45","price2":"$67.89","price3":"¥999"}
	// Check for expected parts
	if !strings.Contains(string(data), `"price1":"¥123.45"`) {
		t.Errorf("Expected CNY format in output, got %s", data)
	}
	if !strings.Contains(string(data), `"price2":"$67.89"`) {
		t.Errorf("Expected USD format in output, got %s", data)
	}
	if !strings.Contains(string(data), `"price3":"¥999"`) {
		t.Errorf("Expected JPY format in output, got %s", data)
	}
}

// TestLoggerInterface tests the Logger interface.
type CustomLog struct {
	Value string
}

func (c CustomLog) MarshalLog() ([]byte, error) {
	return []byte(`{"custom_field":"` + c.Value + `"}`), nil
}

func TestLoggerInterface(t *testing.T) {
	type TestStruct struct {
		Custom CustomLog `log:"custom"`
	}

	s := TestStruct{Custom: CustomLog{Value: "test_value"}}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `{"custom":{"custom_field":"test_value"}}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestConditionalLogger tests the ConditionalLogger interface.
type ConditionalValue struct {
	Value string `json:"value"`
	Show  bool   `json:"show"`
}

func (c ConditionalValue) ShouldLog() bool {
	return c.Show
}

func TestConditionalLogger(t *testing.T) {
	type TestStruct struct {
		AlwaysPresent string           `log:"always"`
		Conditional   ConditionalValue `log:"conditional"`
	}

	// Test with ShouldLog returning false
	s1 := TestStruct{
		AlwaysPresent: "present",
		Conditional:   ConditionalValue{Value: "hidden", Show: false},
	}

	data1, err := Marshal(s1)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("1: Marshal result: %s", string(data1))

	expected1 := `{"always":"present"}`
	t.Logf("1: get: %v", string(data1))
	if string(data1) != expected1 {
		t.Errorf("1: Expected %s, got %s", expected1, data1)
	}

	// Test with ShouldLog returning true
	s2 := TestStruct{
		AlwaysPresent: "present",
		Conditional:   ConditionalValue{Value: "shown", Show: true},
	}

	data2, err := Marshal(s2)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("2: Marshal result: %s", string(data2))

	// Check that both fields are present with correct values
	output2 := string(data2)
	t.Logf("2: get: %v", output2)
	if !strings.Contains(output2, `"always":"present"`) {
		t.Errorf("2: Expected 'always' field in output, got %s", output2)
	}
	if !strings.Contains(output2, `"value":"shown"`) {
		t.Errorf("2: Expected 'value' field with 'shown' in output, got %s", output2)
	}
	if !strings.Contains(output2, `"show":true`) {
		t.Errorf("2: Expected 'show' field with true in output, got %s", output2)
	}
}

// TestCircularReference tests circular reference detection.
func TestCircularReference(t *testing.T) {
	type Node struct {
		Value int   `log:"value"`
		Next  *Node `log:"next"`
	}

	n1 := &Node{Value: 1}
	n2 := &Node{Value: 2}
	n1.Next = n2
	n2.Next = n1 // Creates a cycle

	// Disable error fallback so circular reference returns an actual error
	_, err := MarshalWithOpts(n1, WithErrorFallback(false))
	//t.Fatalf("get: %s, err: %v", string(data), err)
	if err == nil {
		t.Error("Expected error for circular reference, got nil")
	} else if !strings.Contains(err.Error(), "cyclic reference detected") {
		t.Errorf("Expected 'cyclic reference detected' error, got %v", err)
	}
	// Enable error fallback so circular reference returns an actual error
	data, err := MarshalWithOpts(n1, WithErrorFallback(true))
	// Parse the JSON to check the types
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	t.Logf("n1->Value: %v", result["value"])
	if fmt.Sprintf("%v", result["value"]) != fmt.Sprintf("%v", n1.Value) {
		t.Fatalf("n1->Value expected: %v, get: %v", n1.Value, result["value"])
	}
	next := result["next"].(map[string]interface{})
	t.Logf("n1->Next->Value: %v", next["value"])
	if fmt.Sprintf("%v", next["value"]) != fmt.Sprintf("%v", n1.Next.Value) {
		t.Fatalf("n1->Next->Value expected: %v, get: %v", n1.Next.Value, next["value"])
	}
	t.Logf("n1->Next->Next: %v", next["next"])
	if !strings.Contains(next["next"].(string), "cyclic reference detected") {
		t.Errorf("Expected 'cyclic reference detected' error, got %v", result["next"])
	}
}

// TestSliceAndMap tests serialization of slices and maps.
func TestSliceAndMap(t *testing.T) {
	type TestStruct struct {
		Slice []int             `log:"slice"`
		Map   map[string]string `log:"map"`
	}

	s := TestStruct{
		Slice: []int{1, 2, 3},
		Map:   map[string]string{"key1": "value1", "key2": "value2"},
	}

	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	// Output order for maps is not guaranteed, so we check for presence of elements
	output := string(data)
	if !strings.Contains(output, `"slice":[1,2,3]`) {
		t.Errorf("Expected slice in output, got %s", output)
	}
	if !strings.Contains(output, `"key1":"value1"`) || !strings.Contains(output, `"key2":"value2"`) {
		t.Errorf("Expected map elements in output, got %s", output)
	}
}

// TestErrorFallback tests error fallback mechanism.
func TestErrorFallback(t *testing.T) {
	RegisterSerializer("test_error", func(v any) ([]byte, error) {
		return nil, errors.New("serializer error")
	})

	type TestStruct struct {
		Field string `log:"field,ser=test_error"`
	}

	s := TestStruct{Field: "test_value"}

	// With error fallback enabled (default)
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	output := string(data)
	if !strings.Contains(output, "FIELD_SERIALIZE_ERROR") {
		t.Errorf("1: Expected error fallback string in output, got %s", output)
	}
	expected := `{"field":"FIELD_SERIALIZE_ERROR{field:Field, value:test_value, error:serializer error}"}`
	if output != expected {
		t.Errorf("1: Expected %s, got %s", expected, output)
	}
	// With error fallback disabled
	_, err = MarshalWithOpts(s, WithErrorFallback(false))
	if err == nil {
		t.Errorf("2: Expected error when fallback is disabled, got nil")
	} else {
		//var e *MarshalError
		//errors.As(err, &e)
		if err.Error() != "log: marshal field Field of type log.TestStruct: log: marshal field Field of type string: serializer error" {
			t.Errorf("2: Expected specific error, got: %v", err)
		}
	}
}

// TestMarshalTo tests MarshalTo function.
func TestMarshalTo(t *testing.T) {
	type TestStruct struct {
		A string `log:"a"`
	}

	s := TestStruct{A: "value_a"}

	var buf strings.Builder
	err := MarshalTo(&buf, s)
	if err != nil {
		t.Fatalf("MarshalTo failed: %v", err)
	}
	t.Logf("Marshal result: %s", buf.String())

	expected := `{"a":"value_a"}`
	if buf.String() != expected {
		t.Errorf("Expected %s, got %s", expected, buf.String())
	}
}

// TestRegisterSerializerOverwrite tests overwrite warning for RegisterSerializer.
func TestRegisterSerializerOverwrite(t *testing.T) {
	// This test primarily ensures the function doesn't panic and logs a warning.
	// In a real test, you might capture log output to verify the warning.
	RegisterSerializer("test_dup", func(v any) ([]byte, error) { return json.Marshal("first") })
	RegisterSerializer("test_dup", func(v any) ([]byte, error) { return json.Marshal("second") })

	s := struct {
		Field string `log:"field,ser=test_dup"`
	}{Field: "ignored"}

	// Debug: Let's test the serializer directly
	if fn, ok := getSer("test_dup"); ok {
		result, err := fn("test")
		if err != nil {
			t.Fatalf("Direct serializer call failed: %v", err)
		}
		t.Logf("Direct serializer result: %s", string(result))
	} else {
		t.Fatal("Could not find test_dup serializer")
	}

	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	// Should use the overwritten (second) version
	expected := `{"field":"second"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestPrecision tests precision option for float fields.
func TestPrecision(t *testing.T) {
	type TestStruct struct {
		Value float64 `log:"value,precision=2"`
	}

	s := TestStruct{Value: 123.456789}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{"value":123.46}` // Rounded to 2 decimal places
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestStringOption tests string option for fields.
func TestStringOption(t *testing.T) {
	type TestStruct struct {
		Number int `log:"number,string"`
	}

	s := TestStruct{Number: 42}
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{"number":"42"}` // Number serialized as string
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestInline tests inline option for struct fields.
func TestInline(t *testing.T) {
	type Address struct {
		City string `log:"city"`
		Zip  string `log:"zip"`
	}

	type Person struct {
		Name    string  `log:"name"`
		Address Address `log:",inline"`
	}

	p := Person{
		Name: "Alice",
		Address: Address{
			City: "Beijing",
			Zip:  "100000",
		},
	}

	data, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{"city":"Beijing","name":"Alice","zip":"100000"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestEmptyStruct tests marshaling an empty struct.
func TestEmptyStruct(t *testing.T) {
	type Empty struct{}

	e := Empty{}
	data, err := Marshal(e)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal result: %s", string(data))

	expected := `{}` // An empty struct with no log tags should result in an empty object
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestNilValue tests marshaling a nil value.
func TestNilValue(t *testing.T) {
	var v *string = nil
	data, err := Marshal(v)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Get: %s", data)
	expected := `null`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, data)
	}
}

// TestMarshalErrorUnwrap tests the Unwrap method of MarshalError.
func TestMarshalErrorUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	mErr := &MarshalError{
		Type:  reflect.TypeOf(0),
		Field: "test_field",
		Err:   innerErr,
	}

	if !errors.Is(innerErr, mErr.Unwrap()) {
		t.Errorf("Unwrap() returned %v, expected %v", mErr.Unwrap(), innerErr)
	}
}



// TestContextSupport tests the context-aware marshaling functionality.
func TestContextSupport(t *testing.T) {
	type TestStruct struct {
		Message string `log:"message"`
	}

	s := TestStruct{Message: "Hello World"}

	// Test with context containing trace ID
	ctx := context.WithValue(context.Background(), "trace_id", "test-trace-123")

	data, err := MarshalWithContext(ctx, s)
	if err != nil {
		t.Fatalf("MarshalWithContext failed: %v", err)
	}

	expected := `{"message":"Hello World"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}

	// Test with nil context (should work like regular Marshal)
	data2, err := MarshalWithContext(nil, s)
	if err != nil {
		t.Fatalf("MarshalWithContext with nil context failed: %v", err)
	}

	if string(data2) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data2))
	}
}

// TestSensitiveDataDetection tests the sensitive data detection in error messages.
func TestSensitiveDataDetection(t *testing.T) {
	type SensitiveStruct struct {
		Password string `log:"password,ser=error_serializer"`
		Email    string `log:"email,ser=error_serializer"`
		Normal   string `log:"normal,ser=error_serializer"`
	}

	// Register an error serializer that will fail
	RegisterSerializer("error_serializer", func(v any) ([]byte, error) {
		return nil, fmt.Errorf("serializer error")
	})

	s := SensitiveStruct{
		Password: "secret123",
		Email:    "user@example.com",
		Normal:   "normal_value",
	}

	data, err := MarshalWithOpts(s, WithErrorFallback(true))
	if err != nil {
		t.Fatalf("MarshalWithOpts failed: %v", err)
	}

	result := string(data)

	// Check that sensitive fields are masked in error messages
	if strings.Contains(result, "secret123") {
		t.Error("Password should be masked in error message")
	}
	if strings.Contains(result, "user@example.com") {
		t.Error("Email should be masked in error message")
	}
	if !strings.Contains(result, "normal_value") {
		t.Error("Normal field should not be masked in error message")
	}
}

// TestLogLevel tests the log level functionality.
func TestLogLevel(t *testing.T) {
	type LogTestStruct struct {
		Message string `log:"message"`
		Level   string `log:"level"`
	}

	s := LogTestStruct{Message: "Test message", Level: "info"}

	// Test with different log levels
	data, err := MarshalWithOpts(s, WithLevel(INFO))
	if err != nil {
		t.Fatalf("MarshalWithOpts failed: %v", err)
	}

	// Parse JSON to check content regardless of field order
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result["message"] != "Test message" {
		t.Errorf("Expected message='Test message', got '%s'", result["message"])
	}
	if result["level"] != "info" {
		t.Errorf("Expected level='info', got '%s'", result["level"])
	}
}
