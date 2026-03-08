package logger

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

type customActor struct {
	infoBuf  *bytes.Buffer
	errorBuf *bytes.Buffer
}

func (c customActor) Info(msg string)  { c.infoBuf.WriteString(msg) }
func (c customActor) Error(msg string) { c.errorBuf.WriteString(msg) }

func TestLogInitUsesDefaultActorWhenNil(t *testing.T) {
	actor = nil
	config := &LoggerConfig{Level: LogLevelInfo}
	if err := LogInit(config); err != nil {
		t.Fatalf("LogInit returned error: %v", err)
	}
	if actor == nil {
		t.Fatalf("expected default actor when input actor is nil")
	}
}

func TestLogInitUsesProvidedActor(t *testing.T) {
	infoBuf := &bytes.Buffer{}
	errorBuf := &bytes.Buffer{}
	provided := customActor{infoBuf: infoBuf, errorBuf: errorBuf}
	config := &LoggerConfig{Level: LogLevelWarn, Actor: provided}
	if err := LogInit(config); err != nil {
		t.Fatalf("LogInit returned error: %v", err)
	}
	if _, ok := actor.(customActor); !ok {
		t.Fatalf("expected provided actor to be set")
	}
	if logLevel != LogLevelWarn {
		t.Fatalf("expected log level %d, got %d", LogLevelWarn, logLevel)
	}
}

func TestLogInitWithNilConfig(t *testing.T) {
	if err := LogInit(nil); err != nil {
		t.Fatalf("LogInit returned error: %v", err)
	}
	if logLevel != LogLevelVerbose {
		t.Fatalf("expected default log level %d, got %d", LogLevelVerbose, logLevel)
	}
}

func TestWithTimestampPrefix(t *testing.T) {
	out := withTimestamp("hello")
	if !strings.HasPrefix(out, "[") {
		t.Fatalf("expected timestamp prefix, got: %q", out)
	}
	if !strings.Contains(out, "]hello") {
		t.Fatalf("expected suffixed message, got: %q", out)
	}
}

func TestAsyncLogger(t *testing.T) {
	// 测试异步日志基本功能
	config := &LoggerConfig{
		Level:        LogLevelInfo,
		Async:        true,
		AsyncBufSize: 1000,
	}

	err := LogInit(config)
	if err != nil {
		t.Fatalf("LogInit failed: %v", err)
	}

	// 发送一些异步日志
	for i := 0; i < 100; i++ {
		Info("Async log message %d", i)
	}

	// 异步日志应该立即返回，不会阻塞
	// 等待一段时间让后台协程处理部分日志
	time.Sleep(50 * time.Millisecond)

	// 关闭日志系统
	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestAsyncVsSyncPerformance(t *testing.T) {
	// 测试同步日志性能
	syncConfig := &LoggerConfig{
		Level: LogLevelInfo,
		Async: false,
	}

	if err := LogInit(syncConfig); err != nil {
		t.Fatalf("Sync LogInit failed: %v", err)
	}

	start := time.Now()
	for i := 0; i < 100; i++ {
		Info("Sync log message %d", i)
	}
	syncDuration := time.Since(start)

	if err := Shutdown(); err != nil {
		t.Fatalf("Sync Shutdown failed: %v", err)
	}

	// 等待一下避免文件冲突
	time.Sleep(50 * time.Millisecond)

	// 测试异步日志性能
	asyncConfig := &LoggerConfig{
		Level:        LogLevelInfo,
		Async:        true,
		AsyncBufSize: 1000,
	}

	if err := LogInit(asyncConfig); err != nil {
		t.Fatalf("Async LogInit failed: %v", err)
	}

	start = time.Now()
	for i := 0; i < 100; i++ {
		Info("Async log message %d", i)
	}
	asyncDuration := time.Since(start)

	if err := Shutdown(); err != nil {
		t.Fatalf("Async Shutdown failed: %v", err)
	}

	// 输出性能对比
	t.Logf("Sync duration: %v", syncDuration)
	t.Logf("Async duration: %v", asyncDuration)

	// 异步日志应该比同步日志快（只发送到 channel）
	// 注意：实际性能提升幅度取决于磁盘 I/O 性能
	if asyncDuration > syncDuration/2 && syncDuration > 100*time.Millisecond {
		t.Logf("Warning: Async performance benefit limited. Sync: %v, Async: %v", syncDuration, asyncDuration)
	}
}
