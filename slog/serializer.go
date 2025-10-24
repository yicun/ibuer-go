// Package slog provides field-level JSON logging, outputting only fields with the 'slog' tag.
// Priority: Struct Logger → Field Logger → ser=xxx → Basic Type → Mask
package slog

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// ----- Lazy Serializer Registry -----

var (
	lazySerializers   = make(map[string]func() SerializerFunc)
	lazySerializersMu sync.RWMutex
)

// RegisterLazySerializer registers a serializer that will be created on first use
func RegisterLazySerializer(name string, factory func() SerializerFunc) {
	lazySerializersMu.Lock()
	defer lazySerializersMu.Unlock()
	lazySerializers[name] = factory
}

// getLazySerializer gets or creates a lazy serializer
func getLazySerializer(name string) (SerializerFunc, bool) {
	// First check if it's already registered in the main registry
	if fn, ok := getSer(name); ok {
		return fn, true
	}

	// Check lazy registry
	lazySerializersMu.RLock()
	factory, exists := lazySerializers[name]
	lazySerializersMu.RUnlock()

	if !exists {
		return nil, false
	}

	// Create the serializer on first use
	lazySerializersMu.Lock()
	defer lazySerializersMu.Unlock()

	// Double-check after acquiring write lock
	if fn, ok := getSer(name); ok {
		return fn, true
	}

	// Create and register the serializer
	serializerFunc := factory()
	RegisterSerializer(name, serializerFunc)
	delete(lazySerializers, name) // Remove from lazy registry after creation

	return serializerFunc, true
}

// RegisterCurrencyFormattedSerializer registers currency serializers.
func RegisterCurrencyFormattedSerializer() {
	currencyFormats := map[string]struct {
		symbol   string
		decimals int
	}{
		"currency_cny":  {"¥", 2},
		"currency_usd":  {"$", 2},
		"currency_eur":  {"€", 2},
		"currency_gbp":  {"£", 2},
		"currency_jpy":  {"¥", 0},
		"currency_krw":  {"₩", 0},
		"currency_cny4": {"¥", 4},
		"currency_usd4": {"$", 4},
	}

	for name, format := range currencyFormats {
		name := name
		symbol := format.symbol
		decimals := format.decimals

		RegisterLazySerializer(name, func() SerializerFunc {
			return func(v any) ([]byte, error) {
				f, err := toFloat64(v)
				if err != nil {
					return nil, err
				}

				if decimals == 0 {
					return json.Marshal(fmt.Sprintf("%s%d", symbol, int64(f)))
				}

				format := fmt.Sprintf("%s%%.%df", symbol, decimals)
				return json.Marshal(fmt.Sprintf(format, f))
			}
		})
	}

	RegisterLazySerializer("currency", func() SerializerFunc {
		return func(v any) ([]byte, error) {
			return currencySerializer(v, "¥", 2)
		}
	})
}

func currencySerializer(v any, symbol string, decimals int) ([]byte, error) {
	f, err := toFloat64(v)
	if err != nil {
		return nil, err
	}

	if decimals == 0 {
		return json.Marshal(fmt.Sprintf("%s%d", symbol, int64(f)))
	}

	format := fmt.Sprintf("%s%%.%df", symbol, decimals)
	return json.Marshal(fmt.Sprintf(format, f))
}

// RegisterDurationFormattedSerializer registers duration serializers.
func RegisterDurationFormattedSerializer() {
	RegisterLazySerializer("duration", func() SerializerFunc {
		return func(v any) ([]byte, error) {
			if d, ok := v.(time.Duration); ok {
				return json.Marshal(d.Milliseconds())
			}
			return nil, fmt.Errorf("duration serializer expects time.Duration, got %T", v)
		}
	})

	durationFormats := map[string]func(time.Duration) any{
		"duration_ns":      func(d time.Duration) any { return d.Nanoseconds() },
		"duration_us":      func(d time.Duration) any { return d.Microseconds() },
		"duration_ms":      func(d time.Duration) any { return d.Milliseconds() },
		"duration_sec":     func(d time.Duration) any { return d.Seconds() },
		"duration_sec_int": func(d time.Duration) any { return int64(d.Seconds()) },
		"duration_min":     func(d time.Duration) any { return d.Minutes() },
		"duration_hr":      func(d time.Duration) any { return d.Hours() },
		"duration_string":  func(d time.Duration) any { return d.String() },
		"duration_short": func(d time.Duration) any {
			if d < time.Microsecond {
				return fmt.Sprintf("%dns", d.Nanoseconds())
			} else if d < time.Millisecond {
				return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
			} else if d < time.Second {
				return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
			} else if d < time.Minute {
				return fmt.Sprintf("%.2fs", d.Seconds())
			} else if d < time.Hour {
				return fmt.Sprintf("%.2fm", d.Minutes())
			} else {
				return fmt.Sprintf("%.2fh", d.Hours())
			}
		},
		"duration_human": func(d time.Duration) any {
			if d < time.Second {
				return fmt.Sprintf("%d ms", d.Milliseconds())
			}

			var parts []string
			days := d / (24 * time.Hour)
			d -= days * 24 * time.Hour
			hours := d / time.Hour
			d -= hours * time.Hour
			minutes := d / time.Minute
			d -= minutes * time.Minute
			seconds := d / time.Second

			if days > 0 {
				parts = append(parts, fmt.Sprintf("%dd", days))
			}
			if hours > 0 {
				parts = append(parts, fmt.Sprintf("%dh", hours))
			}
			if minutes > 0 {
				parts = append(parts, fmt.Sprintf("%dm", minutes))
			}
			if seconds > 0 || len(parts) == 0 {
				parts = append(parts, fmt.Sprintf("%ds", seconds))
			}

			return strings.Join(parts, " ")
		},
	}

	for name, formatter := range durationFormats {
		formatter := formatter
		RegisterLazySerializer(name, func() SerializerFunc {
			return func(v any) ([]byte, error) {
				if d, ok := v.(time.Duration); ok {
					return json.Marshal(formatter(d))
				}
				return nil, fmt.Errorf("%s serializer expects time.Duration, got %T", name, v)
			}
		})
	}
}

func RegisterDurationSerializerWithPrecision(name string, unit time.Duration, precision int) {
	RegisterLazySerializer(name, func() SerializerFunc {
		return func(v any) ([]byte, error) {
			if d, ok := v.(time.Duration); ok {
				value := float64(d) / float64(unit)
				if precision >= 0 {
					// Round to specified precision and return as float64
					multiplier := math.Pow(10, float64(precision))
					roundedValue := math.Round(value*multiplier) / multiplier
					return json.Marshal(roundedValue)
				}
				return json.Marshal(value)
			}
			return nil, fmt.Errorf("%s serializer expects time.Duration, got %T", name, v)
		}
	})
}

// RegisterTimeFormattedSerializer registers time serializers.
func RegisterTimeFormattedSerializer() {
	RegisterLazySerializer("time_rfc3339", func() SerializerFunc {
		return func(v any) ([]byte, error) {
			if t, ok := v.(time.Time); ok {
				return json.Marshal(t.Format(time.RFC3339))
			}
			return nil, fmt.Errorf("time_rfc3339 serializer expects time.Time, got %T", v)
		}
	})

	timeFormats := map[string]string{
		"time_date":     "2006-01-02",
		"time_datetime": "2006-01-02 15:04:05",
		"time_iso8601":  "2006-01-02T15:04:05Z07:00",
		"time_rfc822":   time.RFC822,
		"time_rfc1123":  time.RFC1123,
		"time_unix":     "",
		"time_unix_ms":  "",
		"time_unix_ns":  "",
		"time_ansic":    time.ANSIC,
		"time_kitchen":  time.Kitchen,
	}

	for name, layout := range timeFormats {
		layout := layout
		RegisterLazySerializer(name, func() SerializerFunc {
			return func(v any) ([]byte, error) {
				if t, ok := v.(time.Time); ok {
					switch name {
					case "time_unix":
						return json.Marshal(t.Unix())
					case "time_unix_ms":
						return json.Marshal(t.UnixMilli())
					case "time_unix_ns":
						return json.Marshal(t.UnixNano())
					default:
						return json.Marshal(t.Format(layout))
					}
				}
				return nil, fmt.Errorf("%s serializer expects time.Time, got %T", name, v)
			}
		})
	}
}

func RegisterTimeSerializerWithLayout(name, layout string) {
	RegisterLazySerializer(name, func() SerializerFunc {
		return func(v any) ([]byte, error) {
			if t, ok := v.(time.Time); ok {
				return json.Marshal(t.Format(layout))
			}
			return nil, fmt.Errorf("%s serializer expects time.Time, got %T", name, v)
		}
	})
}
