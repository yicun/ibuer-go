package sdebug

// SDebuger 调试信息接口定义
type SDebuger interface {
	AddDebugInfo(key string, value any)
	AddDebugInfo2(key1, key2 string, value any)
	SetDebugInfo(key string, debugInfo map[string]any)
	GetDebugInfoMap() map[string]any
	GetDebugInfoStr() string
}

// SDebugInfo 调试信息接口 IDebuger 的默认实现，可嵌入具体 Struct 获取 IDebuger接口能力
type SDebugInfo struct {
	Storage *SDebugStorage `json:"debug_info"`
}

func (d *SDebugInfo) AddDebugInfo(key string, value any) {
	if d.Storage == nil {
		d.Storage = NewDebugInfo(true)
	}
	d.Storage.Set(key, "", value)
}

func (d *SDebugInfo) AddDebugInfo2(key1, key2 string, value any) {
	if d.Storage == nil {
		d.Storage = NewDebugInfo(true)
	}
	d.Storage.Set(key1, key2, value)
}

func (d *SDebugInfo) SetDebugInfo(debugInfo map[string]any) {
	if debugInfo == nil {
		return
	}
	d.Storage = NewDebugInfo(true)
	for k, v := range debugInfo {
		d.Storage.Set(k, "", v)
	}
}

func (d *SDebugInfo) GetDebugInfoMap() map[string]any {
	return d.Storage.ToMap()
}

func (d *SDebugInfo) GetDebugInfoStr() string {
	jsonBytes, _ := d.Storage.ToJSON()
	return string(jsonBytes)
}
