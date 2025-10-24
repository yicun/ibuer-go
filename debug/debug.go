package types

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

// SDebugInfo 调试信息收集器, 提供一个并发安全、只写不读的调试信息收集器.
// - 支持 一级key\两级key 写入，一级key写入Map自动转成二级key
// - 支持 原子计数器、任意值类型(any/[]any/map)
// - 支持 一次性导出 Map 或 Json字节切片
// - 支持 动态开启/关闭调试 模式, 关闭时不写入任何值
// - ToMap / ToJSON 只执行一次，后续调用直接返回缓存值
type SDebugInfo struct {
	enabled   atomic.Bool  // 调试开关
	top       sync.Map     // 一级 key -> 值(map[string]any | *sync.RWMutex)
	mu        sync.RWMutex // 保护 ToMap / ToJSON
	cacheMap  atomic.Value // *map[string]any
	cacheJSON atomic.Value // *[]byte
}

// NewDebugInfo 返回一个新的 SDebugInfo 实例
func NewDebugInfo(enabled bool) *SDebugInfo {
	d := &SDebugInfo{}
	d.enabled.Store(enabled)
	d.cacheMap.Store(&map[string]any{"debug": enabled})
	d.cacheJSON.Store(&[]byte{})
	return d
}

// Set 写入任意值 val
// 若 val 是 map[string]any 且 subKey == ""，则把 val 的 key 作为二级 key
func (d *SDebugInfo) Set(topKey, subKey string, val any) {
	if d.disabled() {
		return
	}
	actual, _ := d.top.LoadOrStore(topKey, make(map[string]any))
	sub := actual.(map[string]any)
	// 获取二级map的锁
	l := d.lockOf(topKey)
	l.Lock()
	defer l.Unlock()
	// val为map 且 subKey == "" 转存 二级key
	if m, ok := val.(map[string]any); ok && subKey == "" {
		for k2, v2 := range m {
			sub[k2] = v2
		}
	} else {
		sub[subKey] = val
	}
}

// Incr 原子累加 delta
func (d *SDebugInfo) Incr(topKey, subKey string, delta int64) {
	if d.disabled() {
		return
	}
	d.atomicCounterOp(topKey, subKey, func(ptr *int64) {
		atomic.AddInt64(ptr, delta)
	})
}

// Store 原子设置计数器
func (d *SDebugInfo) Store(topKey, subKey string, val int64) {
	if d.disabled() {
		return
	}
	d.atomicCounterOp(topKey, subKey, func(ptr *int64) {
		atomic.StoreInt64(ptr, val)
	})
}

// ToMap 返回 Map的深拷贝快照
// 1. 清空所有键值
// 2. 禁用调试模式
// 3. 后续调用直接返回这次缓存值
func (d *SDebugInfo) ToMap() map[string]any {
	// 获取锁
	d.mu.Lock()
	defer d.mu.Unlock()
	// 禁用调试直接返回缓存值
	if d.disabled() {
		if p := d.cacheMap.Load(); p != nil {
			return *p.(*map[string]any)
		} else {
			return map[string]any{"debug": false}
		}
	}
	// 禁止写入，防止并发写
	d.enabled.Store(false)
	// 生成 map
	out := make(map[string]any)
	d.top.Range(func(k, v any) bool {
		// 如为lockKey 直接返回
		if _, ok := k.(lockKey); ok {
			return true
		}
		// 处理正常值
		sub := deepCopyMap(v.(map[string]any))
		switch len(sub) {
		case 0:
			return true
		case 1:
			if val, ok := sub[""]; ok {
				out[k.(string)] = val
				return true
			}
			fallthrough
		default:
			out[k.(string)] = sub
		}
		return true
	})
	// 清空 一级map
	d.top = sync.Map{}
	// 缓存
	d.cacheMap.Store(&out)
	d.cacheJSON.Store(&[]byte{})
	return out
}

// ToJSON 返回 JSON字节切片
// 1. 调用 ToMap 获取键值 并 序列化
// 2. 后续调用直接返回这次缓存值
func (d *SDebugInfo) ToJSON() ([]byte, error) {
	// 判断Json缓存是否为空
	if p := d.cacheJSON.Load(); p != nil {
		if b := *p.(*[]byte); len(b) > 0 {
			return b, nil
		}
	}
	// 获取map
	m := d.ToMap()
	// 获取锁
	d.mu.Lock()
	defer d.mu.Unlock()
	// 二级判断
	if p := d.cacheJSON.Load(); p != nil {
		if b := *p.(*[]byte); len(b) > 0 {
			return b, nil
		}
	}
	// Json序列化
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	d.cacheJSON.Store(&cp)
	return cp, nil
}

// Peek 返回当前调试信息的深拷贝快照（不关闭调试、不缓存、不清空）
// 1. 非flush模式的 ToMap接口
// 2. deepCopyMap可能在大数据量时造成内存压力，慎用！！！
func (d *SDebugInfo) Peek() map[string]any {
	if d.disabled() {
		// 调试已关闭，直接返回缓存值（如果有）
		if p := d.cacheMap.Load(); p != nil {
			return *p.(*map[string]any)
		}
		return map[string]any{"debug": false}
	}

	// 不加写锁，只读遍历
	out := make(map[string]any)
	d.top.Range(func(k, v any) bool {
		// 跳过锁对象
		if _, ok := k.(lockKey); ok {
			return true
		}
		sub := deepCopyMap(v.(map[string]any))
		switch len(sub) {
		case 0:
			return true
		case 1:
			if val, ok := sub[""]; ok {
				out[k.(string)] = val
				return true
			}
			fallthrough
		default:
			out[k.(string)] = sub
		}
		return true
	})
	return out
}

// Clone 返回当前 SDebugInfo 的深拷贝副本，原对象状态（开启/关闭、缓存、数据）全部保持不变
func (d *SDebugInfo) Clone() *SDebugInfo {
	newD := &SDebugInfo{}
	newD.enabled.Store(d.enabled.Load())   // 继承原状态
	newD.cacheMap.Store(&map[string]any{}) // 清空缓存
	newD.cacheJSON.Store(&[]byte{})

	// 只读遍历原对象 d.top
	d.top.Range(func(k, v any) bool {
		// 跳过内部锁 key
		if _, ok := k.(lockKey); ok {
			return true
		}
		topKey := k.(string)
		subMap := deepCopyMap(v.(map[string]any)) // 深拷贝二级 map
		// 写入新实例
		for subKey, val := range subMap {
			newD.Set(topKey, subKey, val)
		}
		return true
	})
	return newD
}

// MarshalJSON 针对encoding/json的自动序列化
func (d *SDebugInfo) MarshalJSON() ([]byte, error) {
	return d.ToJSON()
}

// UnmarshalJSON 针对encoding/json的反自动序列化
func (d *SDebugInfo) UnmarshalJSON(data []byte) error {
	m := map[string]any{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range m {
		d.Set(k, "", v)
	}
	return nil
}

// disabled 返回当前DebugInfo是否被禁用
func (d *SDebugInfo) disabled() bool {
	return !d.enabled.Load()
}

// clearAndDisable 一次性清空所有数据并禁用调试模式
func (d *SDebugInfo) clearAndDisable() {
	d.enabled.Store(false)
	d.top = sync.Map{}
}

// atomicCounterOp 原子计数公共实现
func (d *SDebugInfo) atomicCounterOp(topKey, subKey string, fn func(*int64)) {
	actual, _ := d.top.LoadOrStore(topKey, make(map[string]any))
	sub := actual.(map[string]any)
	// 获取锁
	l := d.lockOf(topKey)
	l.Lock()
	defer l.Unlock()
	// 操作计数器
	var ptr *int64
	if v, ok := sub[subKey]; ok {
		if p, ok := v.(*int64); ok {
			ptr = p
		} else {
			ptr = new(int64)
			sub[subKey] = ptr
		}
	} else {
		ptr = new(int64)
		sub[subKey] = ptr
	}
	fn(ptr)
}

// lockKey 二级map的锁
type lockKey struct{ topKey string }

func (d *SDebugInfo) lockOf(topKey string) *sync.RWMutex {
	lk, _ := d.top.LoadOrStore(lockKey{topKey}, new(sync.RWMutex))
	return lk.(*sync.RWMutex)
}

// deepCopyMap 深拷贝一个 map[string]any
func deepCopyMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var cp map[string]any
	_ = json.Unmarshal(b, &cp)
	return cp
}
