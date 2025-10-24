# log

`log` is a high-performance structured-logging serializer for Go, centered on field-level JSON output.  
Through the `log` tag you decide exactly which fields are emitted, while rich customization—custom serializers,
sensitive-data masking, conditional emission, tracing, levels, etc.—is provided out of the box.

## ✨ New in 2025

- **🔒 Race-free registry** – atomic operations eliminate data races
- **🛡️ Smart sensitive-data guard** – auto-detects & redacts passwords, e-mails, phones, credit-cards …
- **🌐 Context support** – trace/span information flows through `context.Context`
- **📈 Log levels** – built-in severity filtering (`DEBUG` … `FATAL`)
- **💾 Memory friendly** – revamped `sync.Pool` management, zero leaks

## Highlights

- **Field-level control** – only tagged fields (`log:"..."`) are written.
- **High speed** – encoder cached in `sync.Pool`, direct `json.Encoder` writes, no extra allocs.
- **Custom serializers** – register once, use with `ser=name` (time, currency, duration …).
- **Built-in & smart masking** – phone, e-mail, plus auto-detection via keywords/patterns.
- **Cycle detection** – prevents endless recursion on circular structs.
- **Graceful errors** – field failure emits `FIELD_SERIALIZE_ERROR{…}` instead of aborting; sensitive values become
  `<sensitive>`.
- **Conditional output** – implement `ConditionalLogger` to skip whole objects/fields.
- **Struct metadata cache** – reflection info computed once.
- **Fallback to `json.Marshaler`** – optional if no `log` tags present.
- **Thread-safe** – all registrations safe for high-concurrency.
- **Context & level aware** – inject trace IDs, filter by severity.

## Install

```bash
go mod init your-project
go get github.com/yicun/ibuer-slog-go/slog   # replace with real repo
```

Or drop the package locally under `./log` and import it directly.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/yicun/ibuer-log-go/log" // your import path
)

type User struct {
	ID       int       `slog:"id"`
	Name     string    `slog:"name"`
	Email    string    `slog:"email,mask=email"`        // built-in e-mail mask
	Phone    string    `slog:"phone,mask=phone"`        // built-in phone mask
	Created  time.Time `slog:"created_at,ser=time_log"` // custom time layout
	Active   bool      `slog:"is_active"`
	Password string    `slog:"-"` // ignored
}

func main() {
	u := User{
		ID:       12345,
		Name:     "Alice",
		Email:    "alice@example.com",
		Phone:    "13800138000",
		Created:  time.Now(),
		Active:   true,
		Password: "secret123",
	}

	out, _ := log.Marshal(u)
	fmt.Println(string(out))
	// {"id":12345,"name":"Alice","email":"a**e@***.com","phone":"138****8000",
	//  "created_at":"2025/01/01 12:00:05.123","is_active":true}
}
```

## Detailed usage

### 1. Basic tags

```go
type S struct {
A string `slog:"field_a"`
B string `slog:"-"` // skipped
C string `slog:"field_c,omitempty"` // omitted if zero
}
```

### 2. Custom serializer (`ser=`)

```go
type Event struct {
Ts     time.Time `slog:"ts,ser=time_unix_ms"`
Amount float64   `slog:"amount,ser=currency_cny"`
}

e := Event{Ts: time.Now(), Amount: 123.45}
b, _ := log.Marshal(e)
// {"ts":1704067205123,"amount":"¥123.45"}
```

### 3. Masking (`mask=`)

```go
type User struct {
Phone string `slog:"phone,mask=phone"` // 138****8000
Email string `slog:"email,mask=email"` // a**e@***.com
}
```

### 4. Inline structs (`inline`)

```go
type Addr  struct{ City, Zip string `slog:"city,zip"` }
type Person struct {
Name string `slog:"name"`
Addr `slog:",inline"` // flattens fields
}
```

### 5. `Logger` interface

```go
type Custom struct{ V string }

func (c Custom) MarshalLog() ([]byte, error) {
return []byte(`{"custom":"`+c.V+`"}`), nil
}

type Container struct{ Custom Custom `slog:"custom"` }
```

### 6. `ConditionalLogger` interface

```go
type Maybe struct {
Value string
Show  bool
}

func (m Maybe) ShouldLog() bool { return m.Show }

type Data struct {
Always string `slog:"always"`
Maybe Maybe   `slog:"maybe"`
}
```

### 7. Options

```go
out, err := log.MarshalWithOpts(obj,
log.WithIndent("", "  "), // pretty print
log.WithMaskSensitive(false), // disable masking
log.WithErrorFallback(true), // keep going on errors
log.WithLevel(log.INFO), // set severity
)
```

### 8. Register your own

```go
// serializer
log.RegisterSerializer("upper", func (v any) ([]byte, error) {
s, ok := v.(string)
if !ok { return nil, fmt.Errorf("needs string") }
return json.Marshal(strings.ToUpper(s))
})

// masker
log.RegisterMask("first_last", func (s string) string {
if len(s) < 3 { return "***" }
return s[:1] + "***" + s[len(s)-1:]
})
```

### 9. Context support

```go
ctx := context.WithValue(context.Background(), "trace_id", "tx-123")
out, _ := log.MarshalWithContext(ctx, obj)
```

### 10. Log levels

```go
log.MarshalWithOpts(obj, log.WithLevel(log.WARN))
```

## Built-in serializers

**Time**  
`time_rfc3339`, `time_date`, `time_datetime`, `time_iso8601`, `time_rfc822`, `time_rfc1123`, `time_unix`,
`time_unix_ms`, `time_unix_ns`, `time_ansic`, `time_kitchen`, `time_short_date`, `time_long_date`, `time_filename`,
`time_log`

**Duration**  
`duration`, `duration_ns`, `duration_us`, `duration_ms`, `duration_sec`, `duration_sec_int`, `duration_min`,
`duration_hr`, `duration_string`, `duration_short`, `duration_human`, `duration_sec_2`, `duration_sec_3`,
`duration_ms_1`, `duration_min_2`

**Currency**  
`currency_cny`, `currency_usd`, `currency_eur`, `currency_gbp`, `currency_jpy`, `currency_krw`, `currency_cny4`,
`currency_usd4`

## Built-in masks

- `phone` → 138****8000
- `email` → a**e@***.com

## API quick ref

### Core

- `Marshal(v any) ([]byte, error)`
- `MarshalWithOpts(v any, opts ...Option) ([]byte, error)`
- `MarshalTo(w io.Writer, v any, opts ...Option) error`
- `MarshalWithContext(ctx context.Context, v any, opts ...Option) ([]byte, error)`

### Registration (thread-safe)

- `RegisterSerializer(name string, fn SerializerFunc)`
- `RegisterMask(name string, fn MaskFunc)`

### Options

- `WithOptions(opts Options) Option`
- `WithIndent(prefix, indent string) Option`
- `WithMaskSensitive(bool) Option`
- `WithErrorFallback(bool) Option`
- `WithLevel(LogLevel) Option`

### Interfaces

- `Logger`              – `MarshalLog() ([]byte, error)`
- `ConditionalLogger`   – `ShouldLog() bool`

### Types

- `LogLevel`   – `DEBUG, INFO, WARN, ERROR, FATAL`
- `MarshalError` – detailed field-error wrapper

## Error handling & sensitive-data protection

When `WithErrorFallback(true)`:

1. Field errors emit `FIELD_SERIALIZE_ERROR{field:..., value:..., error:...}`
2. Values for fields whose names match sensitive keywords (password, token, ssid …) or detected patterns (e-mail,
   card …) are replaced with `<sensitive>` inside the error message.
3. Serialization continues; the overall JSON remains valid.

## Performance tips

- Re-use struct definitions – metadata is cached.
- Keep custom serializers cheap.
- All registration calls are lock-free reads after first write – safe for hot paths.

## Contributing

Issues & PRs welcome.

## License

MIT
