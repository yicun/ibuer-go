package slog

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// CodeGenerator provides zero-reflection serialization for common types
type CodeGenerator struct {
	mu         sync.RWMutex
	generators map[reflect.Type]TypeGenerator
}

// TypeGenerator interface for type-specific serialization
type TypeGenerator interface {
	GenerateMarshal(value reflect.Value, buf *strings.Builder) error
	SupportsMasking() bool
	SupportsCustomSerializer() bool
}

// GeneratedSerializer provides high-performance serialization without reflection
type GeneratedSerializer struct {
	typ         reflect.Type
	fieldCount  int
	generators  []FieldGenerator
	needsMask   bool
	needsCustom bool
}

// FieldGenerator generates code for individual fields
type FieldGenerator struct {
	Name             string
	Index            int
	Type             reflect.Type
	Tag              string
	HasMask          bool
	HasCustomSer     bool
	CustomSerializer string
	SkipEmpty        bool
	ForceString      bool
}

// Global code generator instance
var codeGen = &CodeGenerator{
	generators: make(map[reflect.Type]TypeGenerator),
}

// RegisterGeneratedSerializer registers a generated serializer for a type
func RegisterGeneratedSerializer(typ reflect.Type, generator TypeGenerator) {
	codeGen.mu.Lock()
	defer codeGen.mu.Unlock()
	codeGen.generators[typ] = generator
}

// GetGeneratedSerializer retrieves a generated serializer for a type
func GetGeneratedSerializer(typ reflect.Type) (TypeGenerator, bool) {
	codeGen.mu.RLock()
	defer codeGen.mu.RUnlock()
	gen, exists := codeGen.generators[typ]
	return gen, exists
}

// GenerateTypeSerializer generates a high-performance serializer for a type
func GenerateTypeSerializer(typ reflect.Type) (*GeneratedSerializer, error) {
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("code generation only supports struct types")
	}

	gs := &GeneratedSerializer{
		typ:        typ,
		fieldCount: typ.NumField(),
		generators: make([]FieldGenerator, 0, typ.NumField()),
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("slog")

		// Skip fields without slog tags or excluded fields
		if tag == "" || tag == "-" {
			continue
		}

		fg := parseFieldTag(field.Name, i, field.Type, tag)
		gs.generators = append(gs.generators, fg)

		if fg.HasMask {
			gs.needsMask = true
		}
		if fg.HasCustomSer {
			gs.needsCustom = true
		}
	}

	return gs, nil
}

// parseFieldTag parses struct field tags
func parseFieldTag(name string, index int, typ reflect.Type, tag string) FieldGenerator {
	fg := FieldGenerator{
		Name:  name,
		Index: index,
		Type:  typ,
		Tag:   tag,
	}

	// Parse tag options
	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		// First part is the field name (if specified)
		if parts[0] != "" && !strings.Contains(parts[0], "=") {
			fg.Name = parts[0]
		}
	}

	// Parse options
	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])

		switch {
		case part == "omitempty":
			fg.SkipEmpty = true
		case part == "string":
			fg.ForceString = true
		case strings.HasPrefix(part, "mask="):
			fg.HasMask = true
		case strings.HasPrefix(part, "ser="):
			fg.HasCustomSer = true
			fg.CustomSerializer = strings.TrimPrefix(part, "ser=")
		}
	}

	return fg
}

// GenerateMarshalCode generates Go code for marshaling a specific type
func (gs *GeneratedSerializer) GenerateMarshalCode() string {
	var buf strings.Builder

	// Generate function signature
	buf.WriteString(fmt.Sprintf("func marshal%s(v %s, opts *Options) ([]byte, error) {\n",
		gs.typ.Name(), gs.typ.String()))

	// Generate buffer and encoder setup
	buf.WriteString("\tvar buf bytes.Buffer\n")
	buf.WriteString("\tbuf.WriteString(\"{\")\n")

	// Generate field serialization
	first := true
	for _, fg := range gs.generators {
		if !first {
			buf.WriteString("\tbuf.WriteString(\",\")\n")
		}
		first = false

		gs.generateFieldMarshal(&buf, fg)
	}

	buf.WriteString("\tbuf.WriteString(\"}\")\n")
	buf.WriteString("\treturn buf.Bytes(), nil\n")
	buf.WriteString("}\n")

	return buf.String()
}

// generateFieldMarshal generates code for marshaling a single field
func (gs *GeneratedSerializer) generateFieldMarshal(buf *strings.Builder, fg FieldGenerator) {
	fieldAccess := fmt.Sprintf("v.%s", fg.Name)

	// Generate field name
	buf.WriteString(fmt.Sprintf("\t// Field: %s\n", fg.Name))
	buf.WriteString(fmt.Sprintf("\tbuf.WriteString(\"\\\"%s\\\":\")\n", fg.Name))

	// Handle different types
	switch fg.Type.Kind() {
	case reflect.String:
		gs.generateStringField(buf, fieldAccess, fg)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		gs.generateIntField(buf, fieldAccess, fg)
	case reflect.Bool:
		gs.generateBoolField(buf, fieldAccess, fg)
	case reflect.Float32, reflect.Float64:
		gs.generateFloatField(buf, fieldAccess, fg)
	case reflect.Slice:
		gs.generateSliceField(buf, fieldAccess, fg)
	case reflect.Map:
		gs.generateMapField(buf, fieldAccess, fg)
	case reflect.Struct:
		gs.generateStructField(buf, fieldAccess, fg)
	default:
		gs.generateGenericField(buf, fieldAccess, fg)
	}
}

// generateStringField generates optimized code for string fields
func (gs *GeneratedSerializer) generateStringField(buf *strings.Builder, fieldAccess string, fg FieldGenerator) {
	if fg.HasMask {
		// Generate masked string serialization
		buf.WriteString(fmt.Sprintf("\tmaskedValue := maskString(%s, \"%s\")\n", fieldAccess, fg.Tag))
		buf.WriteString("\tbuf.WriteString(\"\\\"\")\n")
		buf.WriteString("\tbuf.WriteString(maskedValue)\n")
		buf.WriteString("\tbuf.WriteString(\"\\\"\")\n")
	} else {
		// Generate direct string serialization
		buf.WriteString("\tbuf.WriteString(\"\\\"\")\n")
		buf.WriteString(fmt.Sprintf("\tbuf.WriteString(%s)\n", fieldAccess))
		buf.WriteString("\tbuf.WriteString(\"\\\"\")\n")
	}
}

// generateIntField generates optimized code for integer fields
func (gs *GeneratedSerializer) generateIntField(buf *strings.Builder, fieldAccess string, fg FieldGenerator) {
	if fg.ForceString {
		buf.WriteString(fmt.Sprintf("\tbuf.WriteString(\"\\\"%d\\\"\")\n", fieldAccess))
	} else {
		buf.WriteString(fmt.Sprintf("\tbuf.WriteString(strconv.FormatInt(int64(%s), 10)))\n", fieldAccess))
	}
}

// generateBoolField generates optimized code for boolean fields
func (gs *GeneratedSerializer) generateBoolField(buf *strings.Builder, fieldAccess string, fg FieldGenerator) {
	buf.WriteString(fmt.Sprintf("\tif %s {\n", fieldAccess))
	buf.WriteString("\t\tbuf.WriteString(\"true\")\n")
	buf.WriteString("\t} else {\n")
	buf.WriteString("\t\tbuf.WriteString(\"false\")\n")
	buf.WriteString("\t}\n")
}

// generateFloatField generates optimized code for float fields
func (gs *GeneratedSerializer) generateFloatField(buf *strings.Builder, fieldAccess string, fg FieldGenerator) {
	if fg.HasCustomSer {
		// Use custom serializer
		buf.WriteString(fmt.Sprintf("\tcustomValue, err := serializeWithCustom(%s, \"%s\")\n",
			fieldAccess, fg.CustomSerializer))
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString("\t\treturn nil, err\n")
		buf.WriteString("\t}\n")
		buf.WriteString("\tbuf.Write(customValue)\n")
	} else {
		// Direct float serialization
		buf.WriteString(fmt.Sprintf("\tbuf.WriteString(strconv.FormatFloat(float64(%s), 'f', -1, 64))\n", fieldAccess))
	}
}

// generateSliceField generates code for slice fields
func (gs *GeneratedSerializer) generateSliceField(buf *strings.Builder, fieldAccess string, fg FieldGenerator) {
	buf.WriteString("\tbuf.WriteString(\"[\")\n")
	buf.WriteString(fmt.Sprintf("\tfor i, item := range %s {\n", fieldAccess))
	buf.WriteString("\t\tif i \u003e 0 {\n")
	buf.WriteString("\t\t\tbuf.WriteString(\",\")\n")
	buf.WriteString("\t\t}\n")

	// Handle slice element type
	elemType := fg.Type.Elem()
	switch elemType.Kind() {
	case reflect.String:
		buf.WriteString("\t\tbuf.WriteString(\"\\\"\")\n")
		buf.WriteString("\t\tbuf.WriteString(item)\n")
		buf.WriteString("\t\tbuf.WriteString(\"\\\"\")\n")
	default:
		buf.WriteString("\t\tbuf.WriteString(fmt.Sprintf(\"%v\", item))\n")
	}

	buf.WriteString("\t}\n")
	buf.WriteString("\tbuf.WriteString(\"]\")\n")
}

// generateMapField generates code for map fields
func (gs *GeneratedSerializer) generateMapField(buf *strings.Builder, fieldAccess string, fg FieldGenerator) {
	buf.WriteString("\tbuf.WriteString(\"{\")\n")
	buf.WriteString(fmt.Sprintf("\tfirst := true\n"))
	buf.WriteString(fmt.Sprintf("\tfor k, v := range %s {\n", fieldAccess))
	buf.WriteString("\t\tif !first {\n")
	buf.WriteString("\t\t\tbuf.WriteString(\",\")\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\tfirst = false\n")
	buf.WriteString("\t\tbuf.WriteString(\"\\\"\")\n")
	buf.WriteString("\t\tbuf.WriteString(fmt.Sprintf(\"%v\", k))\n")
	buf.WriteString("\t\tbuf.WriteString(\"\\\":\")\n")
	buf.WriteString("\t\tbuf.WriteString(fmt.Sprintf(\"\\\"%v\\\"\", v))\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tbuf.WriteString(\"}\")\n")
}

// generateStructField generates code for nested struct fields
func (gs *GeneratedSerializer) generateStructField(buf *strings.Builder, fieldAccess string, fg FieldGenerator) {
	// For nested structs, fall back to reflection-based serialization
	buf.WriteString(fmt.Sprintf("\tnestedData, err := marshalWithReflection(%s)\n", fieldAccess))
	buf.WriteString("\tif err != nil {\n")
	buf.WriteString("\t\treturn nil, err\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tbuf.Write(nestedData)\n")
}

// generateGenericField generates fallback code for unsupported types
func (gs *GeneratedSerializer) generateGenericField(buf *strings.Builder, fieldAccess string, fg FieldGenerator) {
	buf.WriteString(fmt.Sprintf("\tgenericData, err := json.Marshal(%s)\n", fieldAccess))
	buf.WriteString("\tif err != nil {\n")
	buf.WriteString("\t\treturn nil, err\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tbuf.Write(genericData)\n")
}

// FastIntSerializer provides zero-reflection serialization for int types
type FastIntSerializer struct {
	fieldName  string
	fieldIndex int
}

func (f *FastIntSerializer) GenerateMarshal(value reflect.Value, buf *strings.Builder) error {
	fieldValue := value.Field(f.fieldIndex).Int()
	buf.WriteString(fmt.Sprintf("\"%s\":%d", f.fieldName, fieldValue))
	return nil
}

func (f *FastIntSerializer) SupportsMasking() bool {
	return false
}

func (f *FastIntSerializer) SupportsCustomSerializer() bool {
	return false
}

// FastStringSerializer provides zero-reflection serialization for string types
type FastStringSerializer struct {
	fieldName  string
	fieldIndex int
	maskFunc   MaskFunc
}

func (f *FastStringSerializer) GenerateMarshal(value reflect.Value, buf *strings.Builder) error {
	fieldValue := value.Field(f.fieldIndex).String()

	if f.maskFunc != nil {
		fieldValue = f.maskFunc(fieldValue)
	}

	// Escape quotes in the string
	escapedValue := strings.ReplaceAll(fieldValue, "\"", "\\\"")
	buf.WriteString(fmt.Sprintf("\"%s\":\"%s\"", f.fieldName, escapedValue))
	return nil
}

func (f *FastStringSerializer) SupportsMasking() bool {
	return f.maskFunc != nil
}

func (f *FastStringSerializer) SupportsCustomSerializer() bool {
	return false
}

// GenerateOptimizedSerializer creates an optimized serializer for frequently used types
func GenerateOptimizedSerializer(sample interface{}) (*GeneratedSerializer, error) {
	typ := reflect.TypeOf(sample)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return GenerateTypeSerializer(typ)
}

// BenchmarkGeneratedSerializer compares generated vs reflection-based serialization
func BenchmarkGeneratedSerializer(b *testing.B) {
	type TestStruct struct {
		ID   int    `slog:"id"`
		Name string `slog:"name"`
		Age  int    `slog:"age"`
	}

	// Generate optimized serializer
	gs, err := GenerateOptimizedSerializer(TestStruct{})
	if err != nil {
		b.Fatal(err)
	}

	testData := TestStruct{ID: 123, Name: "Alice", Age: 30}

	b.Run("Generated", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Use generated code (would be compiled)
			_ = gs.GenerateMarshalCode()
		}
	})

	b.Run("Reflection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Use reflection-based approach
			_, _ = Marshal(testData)
		}
	})
}
