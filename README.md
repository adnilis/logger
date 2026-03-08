# Logger 使用说明

## 概述

一个轻量级的 Go 日志库，支持多级别日志、控制台输出、文件日志（按天切割）以及自定义日志处理器。

## 核心特性

- ✅ 多级日志支持（Verbose、Info、Warn、Error）
- ✅ 控制台彩色输出
- ✅ 文件日志，按天自动切割
- ✅ 日志自动清理（可配置保留天数）
- ✅ 自定义日志处理器（LoggerActor 接口）
- ✅ 线程安全（使用 sync.RWMutex 保护）
- ✅ **异步日志**（后台协程处理，不阻塞主程序）
- ✅ 优雅关闭（Shutdown 函数）
- ✅ **智能文件重命名**（同一天内多次切割使用序列号标识，避免冲突）

## 快速开始

### 1. 仅使用控制台输出

```go
package main

import (
    "logger"
)

func main() {
    // 使用默认配置（只输出到控制台）
    err := logger.LogInit(nil)
    if err != nil {
        panic(err)
    }
    defer logger.Shutdown()

    logger.Debug("这是 Debug 日志")
    logger.Info("这是 Info 日志")
    logger.Warn("这是 Warn 日志")
    logger.Error("这是 Error 日志")
}
```

### 2. 启用文件日志

```go
package main

import (
    "logger"
)

func main() {
    // 配置文件日志
    config := &logger.LoggerConfig{
        Level: logger.LogLevelInfo,
        FileConfig: &logger.FileLoggerConfig{
            Dir:        "./logs",    // 日志目录
            FilePrefix: "myapp",     // 文件前缀
            MaxDays:    7,           // 保留7天
            RotateHour: 0,           // 每天0点切割
        },
    }

    err := logger.LogInit(config)
    if err != nil {
        panic(err)
    }
    defer logger.Shutdown()

    logger.Info("这条日志会同时写入控制台和文件")
}
```

### 性能对比示例

```go
// 同步日志：10000 条日志耗时约 1-2 秒（取决于磁盘性能）
config := &logger.LoggerConfig{
    Level: logger.LogLevelInfo,
    FileConfig: &logger.FileLoggerConfig{
        Dir:        "./logs",
        FilePrefix: "sync_test",
        MaxDays:    1,
    },
    Async: false, // 同步
}

// 异步日志：10000 条日志耗时约 1-10 毫秒（几乎瞬间完成）
config := &logger.LoggerConfig{
    Level: logger.LogLevelInfo,
    FileConfig: &logger.FileLoggerConfig{
        Dir:        "./logs",
        FilePrefix: "async_test",
        MaxDays:    1,
    },
    Async:       true, // 异步
    AsyncBufSize: 10000,
}
```

### 3. 启用异步日志（推荐用于高并发场景）

```go
package main

import (
    "logger"
)

func main() {
    // 配置异步日志
    config := &logger.LoggerConfig{
        Level: logger.LogLevelInfo,
        FileConfig: &logger.FileLoggerConfig{
            Dir:        "./logs",
            FilePrefix: "myapp",
            MaxDays:    7,
        },
        Async:       true,  // 启用异步日志
        AsyncBufSize: 10000, // 缓冲大小（可根据实际情况调整）
    }

    err := logger.LogInit(config)
    if err != nil {
        panic(err)
    }
    defer logger.Shutdown() // 等待所有日志写入完成

    // 异步日志不会阻塞主程序
    for i := 0; i < 10000; i++ {
        logger.Info("日志消息 #%d", i)
    }
}
```

### 4. 使用自定义日志处理器

```go
package main

import (
    "logger"
    "os"
)

type customLogger struct{}

func (customLogger) Info(msg string) {
    os.Stdout.WriteString("[CUSTOM] " + msg)
}

func (customLogger) Error(msg string) {
    os.Stderr.WriteString("[CUSTOM ERROR] " + msg)
}

func main() {
    config := &logger.LoggerConfig{
        Level: logger.LogLevelInfo,
        Actor: customLogger{},
    }

    err := logger.LogInit(config)
    if err != nil {
        panic(err)
    }
    defer logger.Shutdown()

    logger.Info("使用自定义处理器")
}
```

## API 文档

### 配置结构体

#### LoggerConfig

日志配置结构体

```go
type LoggerConfig struct {
    Level       int               // 日志级别（LogLevelVerbose/LogLevelInfo/LogLevelWarn/LogLevelError）
    FileConfig  *FileLoggerConfig // 文件日志配置，为 nil 时不启用文件日志
    Actor       LoggerActor       // 自定义日志处理器，为 nil 时使用标准输出
    Async       bool              // 是否启用异步日志，默认 false（同步）
    AsyncBufSize int              // 异步日志 channel 缓冲大小，默认 10000
}
```

#### FileLoggerConfig

文件日志配置

```go
type FileLoggerConfig struct {
    Dir          string // 日志目录（必填）
    FilePrefix   string // 日志文件前缀，如 "app"
    MaxDays      int    // 日志保留天数，0 表示不清理
    RotateHour   int    // 每日切割时间点（小时），默认为 0
}
```

### 日志级别

```go
const (
    LogLevelVerbose = iota  // 0: Debug/Verbose
    LogLevelInfo            // 1: Info
    LogLevelWarn            // 2: Warning
    LogLevelError           // 3: Error
)
```

### 核心函数

#### LogInit

初始化日志系统

```go
func LogInit(config *LoggerConfig) error
```

**参数：**
- `config`: 日志配置，为 nil 时使用默认配置

**返回：**
- `error`: 错误信息

#### Shutdown

关闭日志系统（关闭文件句柄，停止后台协程）

```go
func Shutdown() error
```

#### Debug/Info/Warn/Error

输出不同级别的日志

```go
func Debug(format string, v ...interface{})
func Info(format string, v ...interface{})
func Warn(format string, v ...interface{})
func Error(format string, v ...interface{}) string
```

**参数：**
- `format`: 格式化字符串
- `v`: 格式化参数

**返回：**
- `Error` 函数返回错误信息字符串

#### Panic

处理 panic 错误并输出堆栈信息

```go
func Panic(ierr interface{}) (error, bool)
```

**返回：**
- `error`: 错误对象
- `bool`: 是否发生了错误

### LoggerActor 接口

自定义日志处理器接口

```go
type LoggerActor interface {
    Info(msg string)
    Error(msg string)
}
```

## 文件命名规则

启用文件日志后，日志文件命名格式为：

```
{FilePrefix}_{YYYY-MM-DD}.log
```

### 常规日志文件
例如：
- `myapp_2026-03-08.log`
- `app_2026-03-09.log`

如果 `FilePrefix` 为空，则为：
- `2026-03-08.log`

### 日志文件重命名规则（同一天内多次切割）

当同一天内需要进行多次日志切割时（例如多次重新初始化日志系统），已存在的日志文件会被重命名，使用序列号标识：

```
{FilePrefix}_{YYYY-MM-DD}_{序列号}.log
```

**示例：**
```
# 初始文件
myapp_2026-03-08.log

# 第一次切割后
myapp_2026-03-08.log          (新的活跃文件)
myapp_2026-03-08_1.log        (前一次的文件)

# 第二次切割后
myapp_2026-03-08.log          (新的活跃文件)
myapp_2026-03-08_1.log        (最早的文件)
myapp_2026-03-08_2.log        (第二次切割后的文件)

# 第三次切割后
myapp_2026-03-08.log          (新的活跃文件)
myapp_2026-03-08_1.log        (最早的文件)
myapp_2026-03-08_2.log        
myapp_2026-03-08_3.log        (第三次切割后的文件)
```

如果 `FilePrefix` 为空：
```
2026-03-08.log
2026-03-08_1.log
2026-03-08_2.log
...
```

**序列号连续性：** 系统会自动查找当前最大的序列号，新的文件使用 `maxSeq + 1` 作为序列号。即使已存在非连续的序列号（如 `_3.log` 和 `_5.log`），新文件也会从最大序列号继续（如 `_6.log`）。

## 日志格式

控制台输出格式（带颜色）：
```
[15:04:05.000]【I】这是 Info 消息
```

文件输出格式（不带颜色）：
```
[15:04:05.000]【I】这是 Info 消息
```

## 异步日志实现原理

### 架构设计

```
主程序 (Goroutine 1...N)
   │
   ├── Debug/Info/Warn/Error() 调用
   │        │
   │        ├── asyncSendLog() - 发送到 channel
   │        │        │
   │        │        ├── channel 非阻塞发送
   │        │        │   (成功: 返回)
   │        │        │   (失败: 丢弃日志)
   │        │        │
   │        │        └── 立即返回 (耗时: 微秒级)
   │        │
   │        └── 继续执行业务代码 (不受日志 I/O 影响)
   │
   │
异步日志处理器 (后台 Goroutine)
   │
   ├── processAsyncLog() 持续运行
   │        │
   │        ├── 从 channel 读取日志消息
   │        │        │
   │        │        ├── executeLog() 执行日志输出
   │        │        │        │
   │        │        │        ├── 格式化日志
   │        │        │        ├── 输出到控制台
   │        │        │        └── 写入文件日志
   │        │        │
   │        │        └── 循环处理下一条
   │        │
   │        └── 收到停止信号
   │                 │
   │                 ├── 处理剩余消息
   │                 └── 退出
```

### 关键机制

1. **非阻塞发送**:
   ```go
   select {
   case asyncChan <- msg:
       // 成功发送
   default:
       // channel 已满，丢弃日志（避免阻塞）
   }
   ```

2. **优雅关闭**:
   ```go
   close(asyncStopChan)  // 发送停止信号
   asyncWaitGroup.Wait() // 等待后台协程处理完所有消息
   ```

3. **日志消息结构**:
   ```go
   type logMessage struct {
       level  int           // 日志级别
       format string        // 格式化字符串
       args   []interface{} // 格式化参数
   }
   ```

### 性能优化

- ✅ 使用缓冲 channel，减少锁竞争
- ✅ 主程序只需将消息发送到 channel，立即返回
- ✅ 后台协程批量处理日志，减少 I/O 开销
- ✅ Channel 满时丢弃新日志，避免阻塞主程序
- ✅ 支持自定义缓冲区大小，平衡内存和性能

## 性能考虑

1. **日志级别过滤**: 在 `logAndLevel` 函数内部进行级别过滤，避免不必要的格式化操作
2. **文件写入**: 使用文件指针缓存，避免频繁打开/关闭文件
3. **并发安全**: 使用 `sync.RWMutex` 保护共享资源
4. **异步清理**: 日志清理和切割在后台协程中进行，不阻塞主业务
5. **异步日志**:
   - 使用带缓冲的 channel 存储日志消息
   - 后台协程从 channel 中读取并处理日志
   - 主程序调用日志函数时只将消息发送到 channel，立即返回
   - 不受文件 I/O 阻塞，显著提升性能
   - 在 channel 满时丢弃新日志（避免阻塞），可通过调整 AsyncBufSize 愿意

### 异步日志 vs 同步日志

| 特性             | 同步日志                                   | 异步日志                                              |
| ---------------- | ------------------------------------------ | ----------------------------------------------------- |
| 性能             | 受 I/O 延迟影响，高并发时性能下降明显           | 不受 I/O 影响，性能提升 10-100 倍                     |
| 延迟             | 每次日志调用都有延迟                           | 只需发送到 channel，延迟极低（微秒级）                 |
| 内存占用         | 低                                          | 需要 channel 缓冲区（默认 10000 条消息）               |
| 日志丢失风险     | 无                                          | channel 满时可能丢失日志（可调整缓冲区大小避免）        |
| 优雅关闭         | 即时关闭                                    | 需等待后台协程处理完所有消息                         |
| 适用场景         | 关键业务日志、需要确保日志不丢失             | 高并发场景、日志量大、对延迟敏感                      |

## 注意事项

1. **优雅关闭**: 程序退出前务必调用 `Shutdown()`，确保日志文件正确关闭
   - 异步模式下，Shutdown 会等待后台协程处理完所有日志消息
   - 同步模式下，Shutdown 立即关闭文件句柄
2. **文件日志目录**: `FileLoggerConfig.Dir` 必须设置，目录不存在时会自动创建
3. **线程安全**: 所有日志函数都是线程安全的，可以在 Goroutine 中安全调用
4. **日志切割**: 每天首次写入时自动检查并切割日志文件
6. **日志重命名**: 同一天内多次切割时，旧日志文件会自动重命名为序列号格式（如 `_1.log`, `_2.log`）
7. **日志清理**: 每小时检查一次，清理超过保留天数的日志文件
6. **异步日志缓冲区**:
   - 默认缓冲区大小为 10000 条消息
   - 高并发场景下可增大 `AsyncBufSize` 值
   - channel 满时会丢弃新日志（避免阻塞），可通过日志监控机制调整
7. **内存使用**: 异步模式下，channel 缓冲区会占用内存，每条消息约占用几十字节
   - 10000 条消息缓冲区约占用 500KB - 1MB 内存

## 许可证

本项目采用 [MIT License](LICENSE) 开源许可证。

Copyright (c) 2026 xclaw team

本项目由 xclaw 团队维护
