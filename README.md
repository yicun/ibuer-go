# ibuer-go - High-Performance Go Libraries

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue.svg)](https://golang.org/doc/go1.23)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/yicun/ibuer-go)](https://goreportcard.com/report/github.com/yicun/ibuer-go)

A collection of **high-performance**, **enterprise-grade** Go libraries designed for **microsecond-level** operations where **every nanosecond counts**.

---

## 🚀 Performance Highlights

| Library | Key Feature | Performance | Memory | Use Case |
|---------|-------------|-------------|---------|----------|
| **SLog** | Structured Logging | ~450ns/op | 0 allocations | Enterprise logging with security |
| **SDebug** | Debug Storage | ~89ns/op | 0 allocations | High-frequency debugging |
| **SCopy** | Deep Copy | ~53ns/op | 48B/op | High-performance object copying |

---

## 📦 Packages

### 🔥 [SLog](./slog) - High-Performance Structured Logging
- **Field-level control** with `log` tags
- **Zero-allocation** serialization
- **Enterprise security** with intelligent data masking
- **7.1x faster** than standard JSON
- Perfect for **microservices** and **high-frequency** applications

### ⚡ [SDebug](./sdebug) - Ultra-High-Performance Debug Storage
- **Nanosecond-level** operations
- **Thread-safe** atomic operations
- **Optional deep copy** with type-specific optimizations
- **1,459x faster** ToMap operations
- Ideal for **trading systems** and **real-time monitoring**

### 🔄 [SCopy](./scopy) - High-Performance Deep Copy
- **Nanosecond-level** primitive copying (~53ns)
- **Complete type support** for all Go types
- **Cycle detection** prevents infinite loops
- **Type caching** for repeated struct operations
- Perfect for **state management** and **data processing**

---

## 🎯 Quick Start

```bash
go get github.com/yicun/ibuer-go/slog
go get github.com/yicun/ibuer-go/sdebug
go get github.com/yicun/ibuer-go/scopy
```

```go
import (
    "github.com/yicun/ibuer-go/slog"
    "github.com/yicun/ibuer-go/sdebug"
    "github.com/yicun/ibuer-go/scopy"
)

// High-performance logging
logger := slog.New(os.Stdout)
logger.Info("User created", user)

// Ultra-fast debug storage
debug := sdebug.NewDebugInfo(true)
debug.Set("user", "id", 123)
debug.Incr("metrics", "requests", 1)

// High-performance deep copy
copier := scopy.New(nil)
copied, err := copier.Copy(complexObject)
```

---

## 🏗️ Architecture

Both libraries are built with **enterprise-grade** architecture:

- **Zero-allocation paths** for common operations
- **Lock-free algorithms** using atomic operations
- **Intelligent caching** with cache-friendly data structures
- **Streaming output** avoiding intermediate buffers
- **Comprehensive error handling** for production reliability

---

## 🛡️ Enterprise Features

### Security First
- **Intelligent data masking** for sensitive information
- **HIPAA/PCI DSS compliance** examples
- **Field-level control** for precise data exposure

### Production Ready
- **Memory management** with configurable limits
- **Observability integration** with metrics and monitoring
- **Container deployment** optimizations
- **Comprehensive testing** with extensive benchmarks

---

## 📊 Performance Comparison

| Operation | Standard Library | IbuER-Go | Improvement |
|-----------|------------------|----------|-------------|
| JSON Marshal | ~3,200ns | ~450ns | **7.1x faster** |
| Debug ToMap | ~9,251ns | ~6.3ns | **1,459x faster** |
| Deep Copy (int) | N/A | ~53ns | **Ultra-fast** |
| Memory Alloc | 1,536B | 0B | **Infinite reduction** |

---

## 🏗️ Use Cases

### High-Frequency Trading
```go
// Nanosecond-level market data tracking
debug.Store("market", "price", update.Price)
debug.Incr("market", "updates", 1)
```

### Microservices
```go
// Distributed tracing with field-level control
logger.WithFields(slog.Fields{
    "service": "order-processor",
    "trace_id": traceID,
}).Info("Order processed", order)
```

### Data Processing
```go
// High-performance state copying
copier := scopy.New(&scopy.Options{EnableCache: true})
copiedState, err := copier.Copy(applicationState)
```

### Healthcare Systems
```go
// HIPAA-compliant data logging
logger.Info("Patient record updated", sanitizedData)
```

---

## 🧪 Testing

```bash
# Run all tests
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run security tests
go test -v -run TestSecurity ./...
```

---

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup
```bash
git clone https://github.com/yicun/ibuer-go.git
cd ibuer-go
go mod download
go test -v ./...
```

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- Built for **maximum performance** in high-concurrency Go applications
- Inspired by the need for **zero-overhead** operations in production systems
- Optimized for **microsecond-level** performance in critical scenarios
- Community-driven development with enterprise focus

---

**IbuER-Go** - Because every nanosecond counts in high-performance systems! 🚀

---

## 📚 额外资源

- [SLog文档](./slog/README.md) - 完整日志指南
- [SDebug文档](./sdebug/README.md) - 调试存储指南
- [SCopy文档](./scopy/README.md) - 深拷贝指南
- [API文档](https://pkg.go.dev/github.com/yicun/ibuer-go)
- [性能指南](./docs/PERFORMANCE.md)
- [安全指南](./docs/SECURITY.md)

For more information, visit our [documentation](https://github.com/yicun/ibuer-go/wiki).

---

# ibuer-go - 高性能Go库

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.23-blue.svg)](https://golang.org/doc/go1.23)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/yicun/ibuer-go)](https://goreportcard.com/report/github.com/yicun/ibuer-go)

一个**高性能**、**企业级**的Go库集合，专为**微秒级**操作设计，其中**每纳秒都至关重要**。

---

## 🚀 性能亮点

| 库 | 关键特性 | 性能 | 内存 | 用例 |
|---------|----------|-------------|----------|-------|
| **SLog** | 结构化日志记录 | ~450ns/次 | 0 分配 | 企业级安全日志 |
| **SDebug** | 调试存储 | ~89ns/次 | 0 分配 | 高频调试 |
| **SCopy** | 深拷贝 | ~53ns/次 | 48B/次 | 高性能对象复制 |

---

## 📦 软件包

### 🔥 [SLog](./slog) - 高性能结构化日志
- **字段级控制**使用`log`标签
- **零分配**序列化
- **企业安全**智能数据屏蔽
- 比标准JSON快**7.1倍**
- 完美适用于**微服务**和**高频**应用

### ⚡ [SDebug](./sdebug) - 超高性能调试存储
- **纳秒级**操作
- **线程安全**原子操作
- **可选深拷贝**类型特定优化
- ToMap操作快**1,459倍**
- 理想用于**交易系统**和**实时监控**

### 🔄 [SCopy](./scopy) - 高性能深拷贝
- **纳秒级**基本类型复制 (~53ns)
- **完整类型支持**所有Go类型
- **循环检测**防止无限循环
- **类型缓存**重复结构体操作优化
- 完美适用于**状态管理**和**数据处理**

---

## 🎯 快速开始

```bash
go get github.com/yicun/ibuer-go/slog
go get github.com/yicun/ibuer-go/sdebug
go get github.com/yicun/ibuer-go/scopy
```

```go
import (
    "github.com/yicun/ibuer-go/slog"
    "github.com/yicun/ibuer-go/sdebug"
    "github.com/yicun/ibuer-go/scopy"
)

// 高性能日志记录
logger := slog.New(os.Stdout)
logger.Info("用户创建", user)

// 超高性能调试存储
debug := sdebug.NewDebugInfo(true)
debug.Set("用户", "id", 123)
debug.Incr("指标", "请求数", 1)

// 高性能深拷贝
copier := scopy.New(nil)
copied, err := copier.Copy(复杂对象)
```

---

## 🏗️ 架构

所有库都使用**企业级**架构构建：

- 常见操作**零分配路径**
- 使用原子操作的**无锁算法**
- **智能缓存**缓存友好数据结构
- **流式输出**避免中间缓冲区
- **全面错误处理**确保生产可靠性

---

## 🛡️ 企业特性

### 安全优先
- **智能数据屏蔽**敏感信息
- **HIPAA/PCI DSS合规**示例
- **字段级控制**精确数据暴露

### 生产就绪
- **内存管理**可配置限制
- **可观测性集成**指标和监控
- **容器部署**优化
- **全面测试**广泛基准测试

---

## 📊 性能对比

| 操作 | 标准库 | IbuER-Go | 改进 |
|-----------|--------------|----------|-------------|
| JSON 序列化 | ~3,200ns | ~450ns | **快7.1倍** |
| 调试转映射 | ~9,251ns | ~6.3ns | **快1,459倍** |
| 深拷贝(int) | N/A | ~53ns | **超快** |
| 内存分配 | 1,536B | 0B | **无限减少** |

---

## 🏗️ 使用案例

### 数据处理
```go
// 高性能状态复制
copier := scopy.New(&scopy.Options{EnableCache: true})
copiedState, err := copier.Copy(applicationState)
```

### 微服务
```go
// 分布式跟踪字段级控制
logger.WithFields(slog.Fields{
    "服务": "订单处理器",
    "跟踪ID": traceID,
}).Info("订单已处理", order)
```

### 高频交易
```go
// 纳秒级市场数据跟踪
debug.Store("市场", "价格", update.Price)
debug.Incr("市场", "更新次数", 1)
```

### 微服务
```go
// 分布式跟踪字段级控制
logger.WithFields(slog.Fields{
    "服务": "订单处理器",
    "跟踪ID": traceID,
}).Info("订单已处理", order)
```

### 医疗系统
```go
// HIPAA合规数据日志记录
logger.Info("患者记录已更新", sanitizedData)
```

---

## 🧪 测试

```bash
# 运行所有测试
go test -v ./...

# 运行基准测试
go test -bench=. -benchmem ./...

# 运行安全测试
go test -v -run TestSecurity ./...
```

---

## 🤝 贡献

我们欢迎贡献！详情请参见[贡献指南](CONTRIBUTING.md)。

### 开发环境设置
```bash
git clone https://github.com/yicun/ibuer-go.git
cd ibuer-go
go mod download
go test -v ./...
```

---

## 📄 许可证

本项目采用MIT许可证 - 详见[LICENSE](LICENSE)文件。

---

## 🙏 致谢

- 为**最大性能**在高并发Go应用中构建
- 受生产系统**零开销**操作需求启发
- 针对关键场景**微秒级**性能优化
- 社区驱动开发，专注企业应用

---

**IbuER-Go** - 因为高性能系统中的每一纳秒都很重要！🚀

---

## 📚 额外资源

- [SLog文档](./slog/README.md) - 完整日志指南
- [SDebug文档](./sdebug/README.md) - 调试存储指南
- [SCopy文档](./scopy/README.md) - 深拷贝指南
- [API文档](https://pkg.go.dev/github.com/yicun/ibuer-go)
- [性能指南](./docs/PERFORMANCE.md)
- [安全指南](./docs/SECURITY.md)

更多信息请访问我们的[文档](https://github.com/yicun/ibuer-go/wiki)。