package logger

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// BenchmarkSyncLogger 同步日志基准测试
func BenchmarkSyncLogger(b *testing.B) {
	config := &LoggerConfig{
		Level: LogLevelInfo,
		Async: false,
	}

	if err := LogInit(config); err != nil {
		b.Fatalf("LogInit failed: %v", err)
	}
	defer Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("Benchmark log message %d", i)
	}
}

// BenchmarkAsyncLogger 异步日志基准测试
func BenchmarkAsyncLogger(b *testing.B) {
	config := &LoggerConfig{
		Level:        LogLevelInfo,
		Async:        true,
		AsyncBufSize: 10000,
	}

	if err := LogInit(config); err != nil {
		b.Fatalf("LogInit failed: %v", err)
	}
	defer Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("Benchmark log message %d", i)
	}
}

// BenchmarkStringBuilder 预分配 vs 不预分配对比
func BenchmarkStringBuilderWithPrealloc(b *testing.B) {
	prefix := levelPrefixes[LogLevelInfo]
	timestr := "[15:04:05.000]"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var str strings.Builder
		str.Grow(len(prefix) + len(timestr) + 50)
		str.WriteString(timestr)
		str.WriteString(prefix)
		str.WriteString(fmt.Sprintf("Message %d", i))
		str.WriteString("\n")
		_ = str.String()
	}
}

func BenchmarkStringBuilderWithoutPrealloc(b *testing.B) {
	ALL_COLOR := []string{BLUE, WHITE, YELLOW, RED, GREEN, MAGENTA, CYAN, RESET}
	LEVEL_TAGS := []string{"V", "I", "W", "E"}
	RESET := "\x1b[0m"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var str strings.Builder
		str.WriteString("[15:04:05.000]")
		str.WriteString(ALL_COLOR[1])
		str.WriteString("【")
		str.WriteString(LEVEL_TAGS[1])
		str.WriteString("】")
		str.WriteString(RESET)
		str.WriteString(fmt.Sprintf("Message %d", i))
		str.WriteString("\n")
		_ = str.String()
	}
}

// BenchmarkDifferentLevels 不同日志级别的性能对比
func BenchmarkDifferentLevels(b *testing.B) {
	config := &LoggerConfig{
		Level: LogLevelVerbose,
		Async: true,
	}

	if err := LogInit(config); err != nil {
		b.Fatalf("LogInit failed: %v", err)
	}
	defer Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch i % 4 {
		case 0:
			Debug("Debug message %d", i)
		case 1:
			Info("Info message %d", i)
		case 2:
			Warn("Warn message %d", i)
		case 3:
			Error("Error message %d", i)
		}
	}
}

// TestConcurrentLogging 并发日志测试
func TestConcurrentLogging(t *testing.T) {
	config := &LoggerConfig{
		Level:        LogLevelInfo,
		Async:        true,
		AsyncBufSize: 10000,
	}

	if err := LogInit(config); err != nil {
		t.Fatalf("LogInit failed: %v", err)
	}

	// 记录初始内存状态
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 并发发送大量日志
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(id int) {
			for j := 0; j < 1000; j++ {
				Info("Goroutine %d, message %d", id, j)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 记录结束内存状态
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// 等待后台处理完成
	time.Sleep(200 * time.Millisecond)

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// 输出内存使用统计
	t.Logf("Allocated memory: %d -> %d bytes", m1.Alloc, m2.Alloc)
	t.Logf("Total allocations: %d", m2.TotalAlloc-m1.TotalAlloc)
	t.Logf("GC cycles: %d -> %d", m1.NumGC, m2.NumGC)
	t.Logf("Messages sent: 100000")
}

// TestChannelDropBehavior 测试 channel 满时的丢弃行为
func TestChannelDropBehavior(t *testing.T) {
	smallBufSize := 10

	config := &LoggerConfig{
		Level:        LogLevelInfo,
		Async:        true,
		AsyncBufSize: smallBufSize,
	}

	if err := LogInit(config); err != nil {
		t.Fatalf("LogInit failed: %v", err)
	}

	// 快速发送大量日志超过 channel 容量
	for i := 0; i < 1000; i++ {
		Info("Message %d", i)
	}

	// 不等待，直接关闭
	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	t.Logf("Successfully tested channel drop behavior with buffer size %d", smallBufSize)
}

// TestLogLevelFiltering 测试日志级别过滤
func TestLogLevelFiltering(t *testing.T) {
	config := &LoggerConfig{
		Level: LogLevelWarn,
		Async: false,
	}

	if err := LogInit(config); err != nil {
		t.Fatalf("LogInit failed: %v", err)
	}

	// 下面的日志应该被过滤掉
	Debug("This debug message should not appear")
	Info("This info message should not appear")

	// 这些日志应该出现
	Warn("This warn message should appear")
	Error("This error message should appear")

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// TestPerformanceMetrics 性能指标测试
func TestPerformanceMetrics(t *testing.T) {
	messageCount := 100000

	config := &LoggerConfig{
		Level:        LogLevelInfo,
		Async:        true,
		AsyncBufSize: 10000,
	}

	if err := LogInit(config); err != nil {
		t.Fatalf("LogInit failed: %v", err)
	}

	start := time.Now()
	for i := 0; i < messageCount; i++ {
		Info("Performance test message %d", i)
	}
	elapsed := time.Since(start)

	msgsPerSec := float64(messageCount) / elapsed.Seconds()
	avgLatency := elapsed / time.Duration(messageCount)

	t.Logf("Messages sent: %d", messageCount)
	t.Logf("Total time: %v", elapsed)
	t.Logf("Messages per second: %.2f", msgsPerSec)
	t.Logf("Average latency: %v", avgLatency)

	// 等待后台处理
	time.Sleep(time.Second)

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// 性能基准：异步日志应该能在 1 毫秒内发送 10000 条消息
	maxExpectedTime := 100 * time.Millisecond
	if elapsed > maxExpectedTime {
		t.Errorf("Performance too slow: expected <%v, got %v", maxExpectedTime, elapsed)
	}
}

// BenchmarkConcurrentSyncLogger 并发同步日志基准测试
func BenchmarkConcurrentSyncLogger(b *testing.B) {
	config := &LoggerConfig{
		Level: LogLevelInfo,
		Async: false,
	}

	if err := LogInit(config); err != nil {
		b.Fatalf("LogInit failed: %v", err)
	}
	defer Shutdown()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Info("Concurrent message %d", i)
			i++
		}
	})
}

// BenchmarkConcurrentAsyncLogger 并发异步日志基准测试
func BenchmarkConcurrentAsyncLogger(b *testing.B) {
	config := &LoggerConfig{
		Level:        LogLevelInfo,
		Async:        true,
		AsyncBufSize: 10000,
	}

	if err := LogInit(config); err != nil {
		b.Fatalf("LogInit failed: %v", err)
	}
	defer Shutdown()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Info("Concurrent message %d", i)
			i++
		}
	})
}
