package sdebug

import "fmt"

// SDebuger 调试信息接口定义
type SDebuger interface {
	AddDebugInfo(key string, value any) error
	AddDebugInfo2(key1, key2 string, value any) error
	SetDebugInfo(debugInfo map[string]any) error
	GetDebugInfoMap() (map[string]any, error)
	GetDebugInfoStr() (string, error)
	GetDebugInfoBytes() ([]byte, error)
}

// SDebugInfo 调试信息接口 IDebuger 的默认实现，可嵌入具体 Struct 获取 IDebuger接口能力
type SDebugInfo struct {
	Storage *SDebugStorage `json:"debug_info"`
}

func (d *SDebugInfo) AddDebugInfo(key string, value any) error {
	if d.Storage == nil {
		d.Storage = NewDebugInfo(true)
	}
	if key == "" {
		return fmt.Errorf("debug key cannot be empty")
	}
	if err := d.Storage.Set(key, "", value); err != nil {
		return fmt.Errorf("failed to set debug info: %w", err)
	}
	return nil
}

func (d *SDebugInfo) AddDebugInfo2(key1, key2 string, value any) error {
	if d.Storage == nil {
		d.Storage = NewDebugInfo(true)
	}
	if key1 == "" {
		return fmt.Errorf("primary debug key cannot be empty")
	}
	if err := d.Storage.Set(key1, key2, value); err != nil {
		return fmt.Errorf("failed to set debug info: %w", err)
	}
	return nil
}

func (d *SDebugInfo) SetDebugInfo(debugInfo map[string]any) error {
	if debugInfo == nil {
		return fmt.Errorf("debug info cannot be nil")
	}
	d.Storage = NewDebugInfo(true)
	for k, v := range debugInfo {
		if err := d.AddDebugInfo(k, v); err != nil {
			return fmt.Errorf("failed to set debug info for key %s: %w", k, err)
		}
	}
	return nil
}

func (d *SDebugInfo) GetDebugInfoMap() (map[string]any, error) {
	if d.Storage == nil {
		return map[string]any{"debug": false}, nil
	}
	return d.Storage.ToMap(), nil
}

func (d *SDebugInfo) GetDebugInfoStr() (string, error) {
	if d.Storage == nil {
		return "{}", nil
	}
	jsonBytes, err := d.Storage.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal debug info to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

func (d *SDebugInfo) GetDebugInfoBytes() ([]byte, error) {
	if d.Storage == nil {
		return []byte("{}"), nil
	}
	return d.Storage.ToJSON()
}
