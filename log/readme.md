# log

`log` 是一个用于 Go 语言的高性能结构化日志序列化库，专注于字段级的 JSON 日志输出。该库允许你通过 `log`
标签精确控制哪些字段被序列化到日志中，并提供了丰富的自定义功能，如自定义序列化器、敏感字段脱敏、条件输出等。

## ✨ 新特性 (2025年更新)

- **🔒 并发安全**：原子操作注册表，消除竞态条件
- **🛡️ 智能敏感数据保护**：自动检测并屏蔽敏感信息（密码、邮箱、手机号等）
- **🌐 上下文支持**：支持链路追踪和上下文信息传递
- **📈 日志级别**：内置日志级别支持，便于日志过滤
- **💾 内存优化**：增强的对象池管理，防止内存泄漏

## 特性

- **字段级控制**：仅序列化带有 `log` 标签的结构体字段。
- **高性能**：使用 `sync.Pool` 缓存编码器，减少内存分配；使用 `json.Encoder` 避免二次序列化。
- **自定义序列化器**：支持为特定类型或字段注册自定义序列化逻辑（如时间格式、货币格式、时长格式等）。
- **敏感字段脱敏**：内置手机号、邮箱脱敏，支持自定义脱敏规则，新增智能敏感数据检测。
- **循环引用检测**：自动检测并防止结构体中的循环引用导致的无限递归。
- **错误回退机制**：当字段序列化失败时，可以选择输出错误信息而非中断整个序列化过程，新增敏感信息保护。
- **条件输出**：支持通过 `ConditionalLogger` 接口实现字段或对象的条件性日志输出。
- **结构体缓存**：缓存结构体字段信息，提升重复序列化的性能。
- **`json.Marshaler` 回退**：如果结构体没有 `log` 标签，可以回退到标准的 `json.Marshal` 行为（可选）。
- **并发安全**：所有注册操作都是线程安全的，支持高并发场景。
- **上下文支持**：支持通过 context 传递链路追踪信息。
- **日志级别**：内置日志级别系统，支持级别过滤。

## 安装

```bash
go mod init your-project-name
go get github.com/your-username/your-repo-name/log # 替换为你的仓库地址
```

或者，如果你将代码放在本地 `log` 目录下，无需 `go get`。

## 快速开始

```go
package main

import (
	"fmt"
	"log"
	"time"
	"your-project-name/log" // 替换为你的导入路径
)

type User struct {
	ID       int       `log:"id"`
	Name     string    `log:"name"`
	Email    string    `log:"email,mask=email"`        // 使用内置邮箱脱敏
	Phone    string    `log:"phone,mask=phone"`        // 使用内置手机号脱敏
	Created  time.Time `log:"created_at,ser=time_log"` // 使用自定义时间序列化器
	Active   bool      `log:"is_active"`
	Password string    `log:"-"` // 忽略此字段
}

func main() {
	user := User{
		ID:       12345,
		Name:     "Alice",
		Email:    "alice@example.com",
		Phone:    "13800138000",
		Created:  time.Now(),
		Active:   true,
		Password: "secret123",
	}

	data, err := log.Marshal(user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output:
	// {"id":12345,"name":"Alice","email":"a**e@***.com","phone":"138****8000","created_at":"2025/01/01 12:00:05.123","is_active":true}
}
```

## 详细用法

### 1. 基本 `log` 标签

```go
type MyStruct struct {
	FieldA string `log:"field_a"`           // 序列化为 "field_a"
	FieldB string `log:"-"`                 // 忽略此字段
	FieldC string `log:"field_c,omitempty"` // 如果值为空，则忽略此字段
}
```

### 2. 自定义序列化器 (`ser=`)

使用 `ser=name` 指定一个已注册的序列化器。

```go
type MyEvent struct {
	Timestamp time.Time `log:"ts,ser=time_unix_ms"`     // Unix 毫秒时间戳
	Amount    float64   `log:"amount,ser=currency_cny"` // 人民币格式
}

event := MyEvent{
	Timestamp: time.Now(),
	Amount:    123.45,
}

data, _ := log.Marshal(event)
fmt.Println(string(data))
// Output: {"ts":1704067205123,"amount":"¥123.45"}
```

### 3. 敏感字段脱敏 (`mask=`)

使用 `mask=name` 指定一个已注册的脱敏规则。

```go
type User struct {
	Phone string `log:"phone,mask=phone"`
	Email string `log:"email,mask=email"`
}

user := User{
	Phone: "13800138000",
	Email: "user@example.com",
}

data, _ := log.Marshal(user)
fmt.Println(string(data))
// Output: {"phone":"138****8000","email":"use***@example.com"}
```

### 4. 内联结构体 (`inline`)

将结构体字段的内部字段直接合并到父对象中。

```go
type Address struct {
	City string `log:"city"`
	Zip  string `log:"zip"`
}

type Person struct {
	Name    string  `log:"name"`
	Address Address `log:",inline"`
}

person := Person{
	Name: "Bob",
	Address: Address{
		City: "Beijing",
		Zip:  "100000",
	},
}

data, _ := log.Marshal(person)
fmt.Println(string(data))
// Output: {"name":"Bob","city":"Beijing","zip":"100000"}
```

### 5. `Logger` 接口

实现 `log.Logger` 接口可以完全自定义序列化逻辑。

```go
type CustomLog struct {
	Value string
}

func (c CustomLog) MarshalLog() ([]byte, error) {
	return []byte(`{"custom_field":"` + c.Value + `"}`), nil
}

type Container struct {
	Custom CustomLog `log:"custom"`
}

container := Container{
	Custom: CustomLog{Value: "hello"},
}

data, _ := log.Marshal(container)
fmt.Println(string(data))
// Output: {"custom":{"custom_field":"hello"}}
```

### 6. `ConditionalLogger` 接口

实现 `log.ConditionalLogger` 接口可以控制对象或字段是否参与序列化。

```go
type ConditionalValue struct {
	Value string
	Show  bool
}

func (c ConditionalValue) ShouldLog() bool {
	return c.Show
}

type Data struct {
	AlwaysPresent string           `log:"always"`
	Conditional   ConditionalValue `log:"conditional"`
}

data := Data{
	AlwaysPresent: "I'm always here",
	Conditional:   ConditionalValue{Value: "I might not appear", Show: false},
}

data2 := Data{
	AlwaysPresent: "I'm always here",
	Conditional:   ConditionalValue{Value: "I will appear", Show: true},
}

d1, _ := log.Marshal(data)
d2, _ := log.Marshal(data2)

fmt.Println(string(d1))
// Output: {"always":"I'm always here"}

fmt.Println(string(d2))
// Output: {"always":"I'm always here","conditional":{"Value":"I will appear","Show":true}}
```

### 7. 序列化选项 (`Options`)

你可以使用 `log.WithOptions` 或其他选项函数来配置序列化行为。

```go
data, err := log.MarshalWithOpts(myStruct,
	log.WithIndent("", "  "),     // 使用 2 个空格缩进
	log.WithMaskSensitive(false), // 禁用默认脱敏
	log.WithErrorFallback(false), // 禁用错误回退，序列化失败时直接返回错误
)
```

### 8. 注册自定义序列化器和脱敏器

```go
// 注册一个自定义的序列化器
log.RegisterSerializer("my_format", func(v any) ([]byte, error) {
	if s, ok := v.(string); ok {
		return json.Marshal(strings.ToUpper(s))
	}
	return nil, fmt.Errorf("my_format only supports string")
})

// 注册一个自定义的脱敏器
log.RegisterMask("my_mask", func(s string) string {
	if len(s) <= 2 {
		return "***"
	}
	return s[:1] + "***" + s[len(s)-1:]
})
```

### 9. 上下文支持

支持通过 context 传递链路追踪信息。

```go
ctx := context.WithValue(context.Background(), "trace_id", "trace-12345")
data, err := log.MarshalWithContext(ctx, myStruct)
```

### 10. 日志级别支持

使用日志级别进行过滤和控制。

```go
data, err := log.MarshalWithOpts(myStruct,
	log.WithLevel(log.INFO), // 设置日志级别
)
```

### 12. 增强的序列化选项

新的序列化选项支持更多功能。

```go
data, err := log.MarshalWithOpts(myStruct,
	log.WithIndent("", "  "),     // 使用 2 个空格缩进
	log.WithMaskSensitive(false), // 禁用默认脱敏
	log.WithErrorFallback(true),  // 启用错误回退，新增敏感信息保护
	log.WithLevel(log.INFO),      // 设置日志级别
)

## 内置序列化器

- `time_rfc3339`, `time_date`, `time_datetime`, `time_iso8601`, `time_rfc822`, `time_rfc1123`, `time_unix`,
  `time_unix_ms`, `time_unix_ns`, `time_ansic`, `time_kitchen`
- `time_short_date`, `time_long_date`, `time_filename`, `time_log`
- `duration`, `duration_ns`, `duration_us`, `duration_ms`, `duration_sec`, `duration_sec_int`, `duration_min`,
  `duration_hr`, `duration_string`, `duration_short`, `duration_human`
- `duration_sec_2`, `duration_sec_3`, `duration_ms_1`, `duration_min_2`
- `currency_cny`, `currency_usd`, `currency_eur`, `currency_gbp`, `currency_jpy`, `currency_krw`, `currency_cny4`,
  `currency_usd4`

## 内置脱敏器

- `phone`: 11位手机号 `138****8000`
- `email`: 邮箱 `a**e@***.com`

## API

### 核心函数
- `Marshal(v any) ([]byte, error)` - 基础序列化
- `MarshalWithOpts(v any, opts ...Option) ([]byte, error)` - 带选项的序列化
- `MarshalTo(w io.Writer, v any, opts ...Option) error` - 写入到 io.Writer
- `MarshalWithContext(ctx context.Context, v any, opts ...Option) ([]byte, error)` - 上下文感知的序列化

### 注册函数
- `RegisterSerializer(name string, fn SerializerFunc)` - 注册自定义序列化器（线程安全）
- `RegisterMask(name string, fn MaskFunc)` - 注册自定义脱敏器（线程安全）

### 选项函数
- `WithOptions(opts Options) Option` - 设置多个选项
- `WithIndent(prefix, indent string) Option` - 设置 JSON 缩进
- `WithMaskSensitive(mask bool) Option` - 设置是否启用敏感字段脱敏
- `WithErrorFallback(enable bool) Option` - 设置是否启用错误回退（新增敏感信息保护）
- `WithLevel(level LogLevel) Option` - 设置日志级别

### 接口
- `Logger` - 自定义序列化接口
- `ConditionalLogger` - 条件输出接口

### 类型
- `LogLevel` - 日志级别类型（DEBUG, INFO, WARN, ERROR, FATAL）
- `MarshalError` - 序列化错误类型

## 错误处理

- `*log.MarshalError`: 包装序列化过程中的具体错误，包含类型和字段信息。

### 敏感信息保护

当启用错误回退 (`WithErrorFallback(true)`) 时，系统会自动检测并保护敏感信息：

- **自动检测**: 识别包含敏感关键词的字段名（如 password, email, phone 等）
- **模式匹配**: 检测邮箱、手机号、信用卡等敏感数据格式
- **智能屏蔽**: 在错误信息中用 `<sensitive>` 替代敏感数据

### 错误回退机制

当字段序列化失败时：

1. 如果启用错误回退，会输出 `FIELD_SERIALIZE_ERROR{field:字段名, value:字段值, error:错误信息}`
2. 敏感字段值会被自动替换为 `<sensitive>`
3. 整个序列化过程继续执行，不会中断

## 性能

`log` 包通过使用 `sync.Pool` 缓存 `encoder` 对象和缓存结构体字段信息来优化性能。对于频繁序列化的场景，性能表现良好。

### 性能优化建议

1. **重用结构体定义**: 结构体信息会被缓存，重复序列化相同类型性能更好
2. **合理使用自定义序列化器**: 避免过度使用复杂的自定义序列化逻辑
3. **并发安全**: 所有注册操作都是线程安全的，可以在高并发场景下安全使用

## 贡献

欢迎提交 Issue 和 Pull Request 来改进 `log` 包。

## 许可证

MIT License

---