package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLogFileRenameOnSameDay 测试同一天已有日志文件时重命名
func TestLogFileRenameOnSameDay(t *testing.T) {
	tempDir := t.TempDir()

	// 创建配置
	config := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:        tempDir,
			FilePrefix: "test",
			MaxDays:    0, // 不自动清理
		},
		Async: false,
	}

	// 第一次初始化
	if err := LogInit(config); err != nil {
		t.Fatalf("First LogInit failed: %v", err)
	}

	// 写入一些日志
	Info("First session log message 1")
	Info("First session log message 2")

	// 关闭
	if err := Shutdown(); err != nil {
		t.Fatalf("First Shutdown failed: %v", err)
	}

	// 检查创建的文件
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(entries))
	}

	firstFileName := entries[0].Name()
	t.Logf("First file created: %s", firstFileName)

	if !strings.HasPrefix(firstFileName, "test_") {
		t.Fatalf("Expected file name to start with 'test_', got: %s", firstFileName)
	}

	// 短暂等待确保文件修改时间不同
	time.Sleep(100 * time.Millisecond)

	// 第二次初始化（同一天，应该重命名旧文件）
	config2 := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:        tempDir,
			FilePrefix: "test",
			MaxDays:    0,
		},
		Async: false,
	}

	if err := LogInit(config2); err != nil {
		t.Fatalf("Second LogInit failed: %v", err)
	}

	// 写入新日志
	Info("Second session log message 1")
	Info("Second session log message 2")

	// 关闭
	if err := Shutdown(); err != nil {
		t.Fatalf("Second Shutdown failed: %v", err)
	}

	// 检查文件
	entries2, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Second ReadDir failed: %v", err)
	}

	t.Logf("Files after second session: %d", len(entries2))

	for _, entry := range entries2 {
		t.Logf("  - %s", entry.Name())
	}

	// 应该有 2 个文件：一个重命名的旧文件，一个新文件
	if len(entries2) < 2 {
		t.Fatalf("Expected at least 2 files (renamed old + new), got %d", len(entries2))
	}

	// 查找新文件（应该是 test_YYYY-MM-DD.log）
	var newFileName string
	var oldFileName string
	today := time.Now().Format("2006-01-02")

	for _, entry := range entries2 {
		name := entry.Name()
		if name == "test_"+today+".log" {
			newFileName = name
		} else if strings.HasPrefix(name, "test_"+today+"_") {
			oldFileName = name
		}
	}

	if newFileName == "" {
		t.Fatal("New file 'test_YYYY-MM-DD.log' not found")
	}

	if oldFileName == "" {
		t.Fatal("Renamed old file not found")
	}

	t.Logf("New file: %s", newFileName)
	t.Logf("Renamed old file: %s", oldFileName)

	// 验证新文件内容包含第二次日志的日志
	newFilePath := filepath.Join(tempDir, newFileName)
	content, err := os.ReadFile(newFilePath)
	if err != nil {
		t.Fatalf("Read new file failed: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Second session log message") {
		t.Errorf("New file should contain second session logs, got: %s", contentStr)
	}
}

// TestMultipleSameDayRotations 测试同一天多次重新初始化
func TestMultipleSameDayRotations(t *testing.T) {
	tempDir := t.TempDir()

	today := time.Now().Format("2006-01-02")

	// 进行 3 次初始化
	for i := 0; i < 3; i++ {
		config := &LoggerConfig{
			Level: LogLevelInfo,
			FileConfig: &FileLoggerConfig{
				Dir:        tempDir,
				FilePrefix: "multi",
				MaxDays:    0,
			},
			Async: false,
		}

		if err := LogInit(config); err != nil {
			t.Fatalf("LogInit attempt %d failed: %v", i+1, err)
		}

		Info("Session %d log message", i+1)

		if err := Shutdown(); err != nil {
			t.Fatalf("Shutdown attempt %d failed: %v", i+1, err)
		}

		time.Sleep(50 * time.Millisecond) // 确保不同的修改时间
	}

	// 检查文件
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	t.Logf("Files created: %d", len(entries))
	for _, entry := range entries {
		t.Logf("  - %s", entry.Name())
	}

	// 应该有 3 个文件
	if len(entries) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(entries))
	}

	// 验证最新文件
	var latestFile string
	for _, entry := range entries {
		if entry.Name() == "multi_"+today+".log" {
			latestFile = entry.Name()
			break
		}
	}

	if latestFile == "" {
		t.Fatal("Latest file not found")
	}

	latestPath := filepath.Join(tempDir, latestFile)
	content, err := os.ReadFile(latestPath)
	if err != nil {
		t.Fatalf("Read latest file failed: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Session 3 log message") {
		t.Errorf("Latest file should contain session 3 logs")
	}
}

// TestLogFileWithoutPrefix 测试不带前缀的日志文件重命名
func TestLogFileWithoutPrefix(t *testing.T) {
	tempDir := t.TempDir()

	config := &LoggerConfig{
		Level: LogLevelInfo,
		FileConfig: &FileLoggerConfig{
			Dir:     tempDir,
			MaxDays: 0,
		},
		Async: false,
	}

	// 第一次初始化
	if err := LogInit(config); err != nil {
		t.Fatalf("First LogInit failed: %v", err)
	}
	Info("Without prefix log 1")
	Shutdown()

	time.Sleep(100 * time.Millisecond)

	// 第二次初始化
	if err := LogInit(config); err != nil {
		t.Fatalf("Second LogInit failed: %v", err)
	}
	Info("Without prefix log 2")
	Shutdown()

	// 检查文件
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) < 2 {
		t.Fatalf("Expected at least 2 files, got %d", len(entries))
	}

	t.Log("Files without prefix:")
	for _, entry := range entries {
		t.Logf("  - %s", entry.Name())
	}

	// 验证有重命名的文件
	today := time.Now().Format("2006-01-02")
	hasTimestampedFile := false
	for _, entry := range entries {
		if strings.Contains(entry.Name(), today+"_") {
			hasTimestampedFile = true
			break
		}
	}

	if !hasTimestampedFile {
		t.Error("Expected to find timestamped renamed file")
	}
}
