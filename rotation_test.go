package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLogFileRotation 同一天内多次初始化测试序列号重命名
func TestLogFileRotation(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 第一次初始化
	config1 := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:        tempDir,
			FilePrefix: "rotate",
			MaxDays:    0,
		},
	}

	if err := LogInit(config1); err != nil {
		t.Fatalf("First LogInit failed: %v", err)
	}

	Info("First batch of logs")
	time.Sleep(100 * time.Millisecond)
	Shutdown()

	// 检查第一次的日志文件
	files1, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(files1) != 1 {
		t.Logf("Warning: Expected 1 file after first init, got %d", len(files1))
	}

	// 第二次初始化（同一天）
	config2 := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:        tempDir,
			FilePrefix: "rotate",
			MaxDays:    0,
		},
	}

	if err := LogInit(config2); err != nil {
		t.Fatalf("Second LogInit failed: %v", err)
	}

	Info("Second batch of logs")
	time.Sleep(100 * time.Millisecond)
	Shutdown()

	// 检查第二次后的日志文件
	files2, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	// 应该有 2 个文件：原来的（已重命名为 _1.log）和新的
	if expected := 2; len(files2) != expected {
		t.Errorf("Expected %d files after second init, got %d", expected, len(files2))
	}

	// 验证文件命名格式
	hasSeq1 := false
	hasCurrent := false
	for _, file := range files2 {
		name := file.Name()
		t.Logf("Found file: %s", name)

		if strings.HasSuffix(name, "_1.log") {
			hasSeq1 = true
		} else if !strings.Contains(name, "_") || strings.HasSuffix(name, fmt.Sprintf("_%s.log", time.Now().Format("2006-01-02"))) {
			hasCurrent = true
		}
	}

	if !hasSeq1 {
		t.Error("Expected file with _1.log suffix (renamed old file)")
	}
	if !hasCurrent {
		t.Error("Expected current day log file")
	}

	// 第三次初始化（同一天）
	config3 := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:        tempDir,
			FilePrefix: "rotate",
			MaxDays:    0,
		},
	}

	if err := LogInit(config3); err != nil {
		t.Fatalf("Third LogInit failed: %v", err)
	}

	Info("Third batch of logs")
	time.Sleep(100 * time.Millisecond)
	Shutdown()

	// 检查第三次后的日志文件
	files3, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	// 应该有 3 个文件
	if expected := 3; len(files3) != expected {
		t.Errorf("Expected %d files after third init, got %d", expected, len(files3))
	}

	// 验证序列号递增
	hasSeq2 := false
	for _, file := range files3 {
		name := file.Name()
		t.Logf("Found file: %s", name)

		if strings.HasSuffix(name, "_2.log") {
			hasSeq2 = true
		}
	}

	if !hasSeq2 {
		t.Error("Expected file with _2.log suffix (second renamed file)")
	}

	t.Logf("Test passed! Found %d log files with sequence numbers:", len(files3))
	for _, file := range files3 {
		t.Logf("  - %s", file.Name())
	}
}

// TestMultipleSameDayRotationsSeq 测试同一天内多次日志切割
func TestMultipleSameDayRotationsSeq(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 初始化日志
	config := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:        tempDir,
			FilePrefix: "daily",
			MaxDays:    0,
		},
	}

	if err := LogInit(config); err != nil {
		t.Fatalf("LogInit failed: %v", err)
	}

	// 写入一些日志
	Info("Initial logs")
	time.Sleep(100 * time.Millisecond)

	// 第一次手动 rotate - 关闭并重新初始化
	Shutdown()

	// 立即重新初始化（同一天）
	if err := LogInit(config); err != nil {
		t.Fatalf("Second LogInit failed: %v", err)
	}

	Info("After first rotation")
	time.Sleep(100 * time.Millisecond)

	// 第二次 rotate
	Shutdown()

	// 第三次初始化
	if err := LogInit(config); err != nil {
		t.Fatalf("Third LogInit failed: %v", err)
	}

	Info("After second rotation")
	time.Sleep(100 * time.Millisecond)
	Shutdown()

	// 检查文件数量和命名
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	t.Logf("Found %d files:", len(files))
	for _, file := range files {
		t.Logf("  - %s", file.Name())
	}

	if expected := 3; len(files) != expected {
		t.Errorf("Expected %d files, got %d", expected, len(files))
	}

	// 验证命名格式
	seq1Found := false
	seq2Found := false
	currentFound := false

	for _, file := range files {
		name := file.Name()
		if name == "daily_2026-03-08_1.log" {
			seq1Found = true
		} else if name == "daily_2026-03-08_2.log" {
			seq2Found = true
		} else if name == "daily_2026-03-08.log" {
			currentFound = true
		} else {
			t.Logf("Unexpected file: %s", name)
		}
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, "_1.log") {
			seq1Found = true
		} else if strings.HasSuffix(name, "_2.log") {
			seq2Found = true
		} else if !strings.Contains(name, "_") {
			currentFound = true
		}
	}

	if !seq1Found {
		t.Error("Expected daily_2026-03-08_1.log to exist")
	}
	if !seq2Found {
		t.Error("Expected daily_2026-03-08_2.log to exist")
	}
	if !currentFound {
		t.Error("Expected daily_2026-03-08.log to exist (current)")
	}

	t.Log("Multiple same-day rotation test passed!")
}

// TestRotationNoPrefix 测试没有前缀的日志文件旋转
func TestRotationNoPrefix(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 第一次初始化（无前缀）
	config1 := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:     tempDir,
			MaxDays: 0,
		},
	}

	if err := LogInit(config1); err != nil {
		t.Fatalf("First LogInit failed: %v", err)
	}

	Info("First logs without prefix")
	time.Sleep(100 * time.Millisecond)
	Shutdown()

	// 第二次初始化
	config2 := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:     tempDir,
			MaxDays: 0,
		},
	}

	if err := LogInit(config2); err != nil {
		t.Fatalf("Second LogInit failed: %v", err)
	}

	Info("Second logs without prefix")
	time.Sleep(100 * time.Millisecond)
	Shutdown()

	// 检查文件
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if expected := 2; len(files) != expected {
		t.Errorf("Expected %d files, got %d", expected, len(files))
	}

	// 验证命名格式（无前缀）
	foundSeq1 := false
	for _, file := range files {
		name := file.Name()
		t.Logf("File: %s", name)

		// 无前缀格式: 2026-03-08_1.log
		if strings.HasSuffix(name, "_1.log") {
			foundSeq1 = true
		}
	}

	if !foundSeq1 {
		t.Error("Expected file with _1.log suffix (no prefix rotation)")
	}

	t.Log("Rotation without prefix test passed!")
}

// TestSequenceNumberContinuity 测试序列号的连续性
func TestSequenceNumberContinuity(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建一些预存在的带序列号的文件
	// 创建 _3.log 和 _5.log，模拟之前已经被重命名过的文件
	// 还需要创建当前活跃的日志文件（不带序列号）
	today := time.Now().Format("2006-01-02")
	currentFile := filepath.Join(tempDir, fmt.Sprintf("continuity_%s.log", today))
	file3 := filepath.Join(tempDir, fmt.Sprintf("continuity_%s_3.log", today))
	file5 := filepath.Join(tempDir, fmt.Sprintf("continuity_%s_5.log", today))

	if err := os.WriteFile(file3, []byte("existing file 3"), 0644); err != nil {
		t.Fatalf("Create test file 3 failed: %v", err)
	}
	if err := os.WriteFile(file5, []byte("existing file 5"), 0644); err != nil {
		t.Fatalf("Create test file 5 failed: %v", err)
	}
	// 创建当前活跃的日志文件（不带序列号），这会触发 rotate 逻辑
	if err := os.WriteFile(currentFile, []byte("current active file"), 0644); err != nil {
		t.Fatalf("Create current file failed: %v", err)
	}

	// 现在初始化日志，应该找到最大序列号是 5
	config := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:        tempDir,
			FilePrefix: "continuity",
			MaxDays:    0,
		},
	}

	if err := LogInit(config); err != nil {
		t.Fatalf("LogInit failed: %v", err)
	}

	Info("Test log")
	time.Sleep(100 * time.Millisecond)
	Shutdown()

	// 检查文件
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	t.Logf("Found %d files:", len(files))
	for _, file := range files {
		t.Logf("  - %s", file.Name())
	}

	// 应该有 4 个文件：旧的 _3.log, 旧的 _5.log, 旧的 _6.log (之前的文件被重命名), 当前活跃文件
	expectedFiles := 4
	if len(files) != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, len(files))
	}

	// 验证新文件是 _6.log
	foundSeq6 := false
	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, "_6.log") {
			foundSeq6 = true
		}
	}

	if !foundSeq6 {
		t.Error("Expected new file to be _6.log (continuing from max seq 5)")
	}
}
