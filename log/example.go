// Package log_test provides examples for the log package.
package log

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// User 示例结构体，展示了多种 log 标签用法
type User struct {
	ID       int       `log:"id"`
	Name     string    `log:"name"`
	Email    string    `log:"email,mask=email"`        // 使用内置邮箱脱敏
	Phone    string    `log:"phone,mask=phone"`        // 使用内置手机号脱敏
	Created  time.Time `log:"created_at,ser=time_log"` // 使用自定义时间序列化器
	Active   bool      `log:"is_active"`
	Password string    `log:"-"` // 忽略此字段
	// 嵌套结构体示例
	Address Address `log:"address,inline"` // 内联地址字段
}

type Address struct {
	City string `log:"city"`
	Zip  string `log:"zip"`
}

// APIResponse 示例结构体，展示 Logger 接口用法
type APIResponse struct {
	Code    int
	Message string
	Data    any
}

// MarshalLog 实现 log.Logger 接口，自定义序列化逻辑
func (r APIResponse) MarshalLog() ([]byte, error) {
	// 构建自定义 JSON 结构
	custom := map[string]any{
		"status_code": r.Code,
		"message":     r.Message,
		"payload":     r.Data,
	}
	return json.Marshal(custom)
}

// ConditionalUser 示例结构体，展示 ConditionalLogger 接口用法
type ConditionalUser struct {
	Name string
	Show bool
}

// ShouldLog 实现 log.ConditionalLogger 接口
func (u ConditionalUser) ShouldLog() bool {
	return u.Show
}

func ExampleMarshal() {
	user := User{
		ID:       12345,
		Name:     "Alice",
		Email:    "alice@example.com",
		Phone:    "13800138000",
		Created:  time.Date(2025, 1, 1, 12, 0, 5, 123000000, time.UTC),
		Active:   true,
		Password: "secret123", // 这个字段会被忽略
		Address: Address{
			City: "Beijing",
			Zip:  "100000",
		},
	}

	data, err := Marshal(user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output:
	// {"id":12345,"name":"Alice","email":"a**e@***.com","phone":"138****8000","created_at":"2025/01/01 12:00:05.123","is_active":true,"city":"Beijing","zip":"100000"}
}

func ExampleMarshalWithOpts() {
	user := User{
		ID:       0, // 空值
		Name:     "Bob",
		Email:    "",
		Phone:    "13900139000",
		Created:  time.Now(),
		Active:   false, // 空值
		Password: "another_secret",
		Address: Address{
			City: "Shanghai",
			Zip:  "200000",
		},
	}

	// 使用 OmitEmptyByDefault 选项，忽略零值字段
	data, err := MarshalWithOpts(user, WithOptions(Options{
		OmitEmptyByDefault: true,
	}))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output: (ID, Email, Active 会被忽略)
	// {"name":"Bob","phone":"139****9000","created_at":"2025/01/01 12:00:05.123","city":"Shanghai","zip":"200000"}
}

func ExampleMarshalWithIndent() {
	user := User{
		ID:       67890,
		Name:     "Charlie",
		Email:    "charlie@test.org",
		Phone:    "15600156000",
		Created:  time.Now(),
		Active:   true,
		Password: "no_log",
		Address: Address{
			City: "Guangzhou",
			Zip:  "510000",
		},
	}

	// 使用缩进格式化 JSON
	data, err := MarshalWithOpts(user, WithIndent("", "  "))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output: (格式化后的 JSON)
	// {
	//   "id": 67890,
	//   "name": "Charlie",
	//   "email": "c**r@***.org",
	//   "phone": "156****6000",
	//   "created_at": "2025/01/01 12:00:05.123",
	//   "is_active": true,
	//   "city": "Guangzhou",
	//   "zip": "510000"
	// }
}

func ExampleCustomSerializer() {
	type Event struct {
		Timestamp time.Time     `log:"ts,ser=time_unix_ms"`     // Unix 毫秒时间戳
		Duration  time.Duration `log:"dur,ser=duration_sec_2"`  // 秒，保留两位小数
		Amount    float64       `log:"amount,ser=currency_usd"` // 美元格式
	}

	event := Event{
		Timestamp: time.Date(2025, 1, 1, 10, 30, 0, 0, time.UTC),
		Duration:  3*time.Minute + 45*time.Second,
		Amount:    123.456,
	}

	data, err := Marshal(event)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output: {"ts":1730484600000,"dur":195.00,"amount":"$123.46"}
}

func ExampleLoggerInterface() {
	resp := APIResponse{
		Code:    200,
		Message: "Success",
		Data:    map[string]string{"key": "value"},
	}

	data, err := Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output: {"status_code":200,"message":"Success","payload":{"key":"value"}}
}

func ExampleConditionalLogger() {
	conditionalUser := ConditionalUser{
		Name: "David",
		Show: false, // 设置为 false，ShouldLog() 返回 false
	}

	// 这个对象会被完全忽略
	data, err := Marshal(conditionalUser)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output: null

	conditionalUser.Show = true // 设置为 true
	data, err = Marshal(conditionalUser)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output: {"Name":"David","Show":true}
}

func ExampleRegisterSerializer() {
	// 注册一个自定义序列化器，将字符串转为大写
	RegisterSerializer("upper", func(v any) ([]byte, error) {
		if s, ok := v.(string); ok {
			return json.Marshal(s + " (UPPERCASED)")
		}
		return nil, fmt.Errorf("upper serializer expects string, got %T", v)
	})

	type Custom struct {
		Text string `log:"text,ser=upper"`
	}

	c := Custom{Text: "hello world"}
	data, err := Marshal(c)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output: {"text":"hello world (UPPERCASED)"}
}

func ExampleRegisterMask() {
	// 注册一个自定义脱敏器，只保留第一个和最后一个字符
	RegisterMask("first_last", func(s string) string {
		if len(s) <= 2 {
			return "***"
		}
		return s[:1] + "***" + s[len(s)-1:]
	})

	type Custom struct {
		SSN string `log:"ssn,mask=first_last"`
	}

	c := Custom{SSN: "123456789"}
	data, err := Marshal(c)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
	// Output: {"ssn":"1***9"}
}
