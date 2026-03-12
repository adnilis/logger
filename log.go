package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	LogLevelVerbose = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

const (
	// 默认异步日志 channel 缓冲大小
	defaultAsyncBufferSize = 10000
)

var (
	LogLevelTags     = []string{"V", "I", "W", "E"}
	logLevel         = LogLevelVerbose
	actor            LoggerActor
	globalFileLogger *fileLogger
	fileMux          sync.RWMutex

	// 异步日志相关
	asyncEnabled   bool
	asyncChan      chan *logMessage
	asyncWaitGroup sync.WaitGroup
	asyncStopChan  chan struct{}

	// 性能优化：预计算颜色前缀
	levelPrefixes []string
)

func init() {
	// 预计算各级别的颜色前缀（不包含 RESET，让整个日志行都有颜色）
	levelPrefixes = make([]string, 4)
	for i := 0; i < 4; i++ {
		levelPrefixes[i] = ALL_COLOR[i] + "【" + LogLevelTags[i] + "】"
	}
}

type stdLoggerActor struct{}

func (stdLoggerActor) Info(msg string) {
	_, _ = os.Stdout.WriteString(withTimestamp(msg))
}

func (stdLoggerActor) Error(msg string) {
	_, _ = os.Stderr.WriteString(withTimestamp(msg))
}

func withTimestamp(msg string) string {
	return "[" + time.Now().Format("15:04:05.000") + "]" + msg
}

var (
	GREEN     = string([]byte{27, 91, 51, 50, 109})
	WHITE     = string([]byte{27, 91, 51, 55, 109})
	YELLOW    = string([]byte{27, 91, 51, 51, 109})
	RED       = string([]byte{27, 91, 51, 49, 109})
	BLUE      = string([]byte{27, 91, 51, 52, 109})
	MAGENTA   = string([]byte{27, 91, 51, 53, 109})
	CYAN      = string([]byte{27, 91, 51, 54, 109})
	RESET     = string([]byte{27, 91, 48, 109})
	ALL_COLOR = []string{BLUE, WHITE, YELLOW, RED, GREEN, MAGENTA, CYAN, RESET}
)

func SetLogLevel(level int) {
	logLevel = level
}

func IsDebug() bool {
	return logLevel == LogLevelVerbose
}

// LoggerConfig 日志配置结构体
type LoggerConfig struct {
	Level        int               // 日志级别
	FileConfig   *FileLoggerConfig // 文件日志配置，为 nil 时不启用文件日志
	Actor        LoggerActor       // 自定义日志处理器，为 nil 时使用标准输出
	Async        bool              // 是否启用异步日志，默认 false（同步）
	AsyncBufSize int               // 异步日志 channel 缓冲大小，默认 10000
}

// FileLoggerConfig 文件日志配置
type FileLoggerConfig struct {
	Dir        string // 日志目录
	FilePrefix string // 日志文件前缀，如 "app"
	MaxDays    int    // 日志保留天数，0 表示不清理
	RotateHour int    // 每日切割时间点（小时），默认为 0
}

// 日志初始化
func LogInit(config *LoggerConfig) error {
	if config == nil {
		config = &LoggerConfig{Level: LogLevelVerbose}
	}

	// 设置日志处理器
	if config.Actor == nil {
		actor = stdLoggerActor{}
	} else {
		actor = config.Actor
	}
	logLevel = config.Level

	// 初始化文件日志
	if config.FileConfig != nil {
		fileMux.Lock()
		defer fileMux.Unlock()

		var err error
		globalFileLogger, err = newFileLogger(config.FileConfig)
		if err != nil {
			return fmt.Errorf("init file logger failed: %w", err)
		}
	}

	// 初始化异步日志
	if config.Async {
		bufSize := config.AsyncBufSize
		if bufSize <= 0 {
			bufSize = defaultAsyncBufferSize
		}
		startAsyncLogger(bufSize)
	}

	return nil
}

type LoggerActor interface {
	Info(msg string)
	Error(msg string)
}

// logMessage 异步日志消息
type logMessage struct {
	level  int
	format string
	args   []interface{}
}

// startAsyncLogger 启动异步日志消费者
func startAsyncLogger(bufSize int) {
	asyncChan = make(chan *logMessage, bufSize)
	asyncStopChan = make(chan struct{})
	asyncEnabled = true

	asyncWaitGroup.Go(func() {
		processAsyncLog()
	})
}

// processAsyncLog 异步日志处理器
func processAsyncLog() {
	for {
		select {
		case msg := <-asyncChan:
			executeLog(msg.level, msg.format, msg.args...)
		case <-asyncStopChan:
			// 处理剩余消息
			for len(asyncChan) > 0 {
				msg := <-asyncChan
				executeLog(msg.level, msg.format, msg.args...)
			}
			return
		}
	}
}

// executeLog 实际执行日志输出（优化版）
func executeLog(level int, format string, v ...interface{}) {
	if level < logLevel {
		return
	}

	s := fmt.Sprintf(format, v...)
	prefix := levelPrefixes[level]
	timestr := "[" + time.Now().Format("15:04:05.000") + "]"

	var str strings.Builder
	str.Grow(len(s) + len(prefix) + len(timestr) + 2)

	if actor == nil {
		str.WriteString(timestr)
		str.WriteString(prefix)
		str.WriteString(s)
		str.WriteString("\n")
		fmt.Fprint(os.Stdout, str.String())
		return
	}

	str.WriteString(prefix)
	str.WriteString(s)
	str.WriteString("\n")

	if level == LogLevelError {
		actor.Error(str.String())
	} else {
		actor.Info(str.String())
	}

	// 写入文件日志（无颜色）
	if globalFileLogger != nil {
		var plainStr strings.Builder
		plainStr.Grow(len(s) + len(LogLevelTags[level]) + len(timestr) + 6)
		plainStr.WriteString(timestr)
		plainStr.WriteString("【")
		plainStr.WriteString(LogLevelTags[level])
		plainStr.WriteString("】")
		plainStr.WriteString(s)
		plainStr.WriteString("\n")
		globalFileLogger.write(plainStr.String())
	}
}

// fileLogger 文件日志器
type fileLogger struct {
	config      *FileLoggerConfig
	currentFile *os.File
	currentDate string
	mu          sync.Mutex
	closed      bool
}

func newFileLogger(config *FileLoggerConfig) (*fileLogger, error) {
	if config.Dir == "" {
		return nil, fmt.Errorf("log dir cannot be empty")
	}

	// 确保日志目录存在
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir failed: %w", err)
	}

	fl := &fileLogger{
		config: config,
	}

	// 初始化日志文件
	if err := fl.rotate(); err != nil {
		return nil, err
	}

	// 启动定时清理和切割协程
	go fl.runBackgroundTasks()

	return fl, nil
}

func (fl *fileLogger) write(msg string) {
	if fl == nil || fl.closed {
		return
	}

	fl.mu.Lock()
	defer fl.mu.Unlock()

	// 检查是否需要切割
	now := time.Now()
	today := now.Format("2006-01-02")
	if today != fl.currentDate {
		if err := fl.rotate(); err != nil {
			fmt.Fprintf(os.Stderr, "rotate log file failed: %v\n", err)
			return
		}
	}

	if fl.currentFile != nil {
		fl.currentFile.WriteString(msg)
	}
}

// rotate 切割日志文件，重命名已存在的同日期日志
func (fl *fileLogger) rotate() error {
	now := time.Now()
	today := now.Format("2006-01-02")

	// 关闭旧文件
	if fl.currentFile != nil {
		_ = fl.currentFile.Close()
	}

	// 构建新日志文件名
	var filename string
	if fl.config.FilePrefix != "" {
		filename = fmt.Sprintf("%s_%s.log", fl.config.FilePrefix, today)
	} else {
		filename = fmt.Sprintf("%s.log", today)
	}

	fullPath := filepath.Join(fl.config.Dir, filename)

	// 检查文件是否已存在
	if _, err := os.Stat(fullPath); err == nil {
		// 文件已存在，重命名为序列号
		oldFilePath := fullPath

		// 查找当前最大的序列号
		maxSeq := 0
		var pattern string
		if fl.config.FilePrefix != "" {
			pattern = fmt.Sprintf("%s_%s_*.log", fl.config.FilePrefix, today)
		} else {
			pattern = fmt.Sprintf("%s_*.log", today)
		}

		matches, _ := filepath.Glob(filepath.Join(fl.config.Dir, pattern))

		for _, match := range matches {
			// 从文件名中提取序列号
			// 格式: app_2026-03-08_1.log -> seq=1
			base := filepath.Base(match)
			var seq int
			if fl.config.FilePrefix != "" {
				fmt.Sscanf(base, fmt.Sprintf("%s_%s_%%d.log", fl.config.FilePrefix, today), &seq)
			} else {
				fmt.Sscanf(base, fmt.Sprintf("%s_%%d.log", today), &seq)
			}
			if seq > maxSeq {
				maxSeq = seq
			}
		}

		// 新的序列号 = 当前最大序列号 + 1
		newSeq := maxSeq + 1

		// 构建重命名后的文件名（统一使用序列号）
		var renamedFilename string
		if fl.config.FilePrefix != "" {
			renamedFilename = fmt.Sprintf("%s_%s_%d.log", fl.config.FilePrefix, today, newSeq)
		} else {
			renamedFilename = fmt.Sprintf("%s_%d.log", today, newSeq)
		}
		renamedFilePath := filepath.Join(fl.config.Dir, renamedFilename)

		// 重命名文件
		if err := os.Rename(oldFilePath, renamedFilePath); err != nil {
			// 如果重命名失败（例如文件被占用），尝试追加到原文件
			fmt.Fprintf(os.Stderr, "rename old log file failed: %v, will append to existing file\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "renamed old log file: %s -> %s\n", oldFilePath, renamedFilename)
		}
	}

	// 创建新日志文件
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}

	fl.currentFile = file
	fl.currentDate = today

	return nil
}

// runBackgroundTasks 运行后台任务：日志切割和清理
func (fl *fileLogger) runBackgroundTasks() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fl.mu.Lock()
			now := time.Now()

			// 检查是否需要切割
			today := now.Format("2006-01-02")
			if today != fl.currentDate {
				if err := fl.rotate(); err != nil {
					fmt.Fprintf(os.Stderr, "rotate log file failed: %v\n", err)
				}
			}

			// 清理过期日志
			if fl.config.MaxDays > 0 {
				fl.cleanOldLogs(now)
			}

			fl.mu.Unlock()
		default:
		}

		if fl.closed {
			return
		}
	}
}

// cleanOldLogs 清理过期日志文件
func (fl *fileLogger) cleanOldLogs(now time.Time) {
	entries, err := os.ReadDir(fl.config.Dir)
	if err != nil {
		return
	}

	cutoffTime := now.AddDate(0, 0, -fl.config.MaxDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 检查文件扩展名
		if filepath.Ext(entry.Name()) != ".log" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 如果文件修改时间早于截止时间，删除它
		if info.ModTime().Before(cutoffTime) {
			filePath := filepath.Join(fl.config.Dir, entry.Name())
			_ = os.Remove(filePath)
		}
	}
}

// close 关闭文件日志器
func (fl *fileLogger) close() error {
	if fl == nil {
		return nil
	}

	fl.mu.Lock()
	defer fl.mu.Unlock()

	fl.closed = true
	if fl.currentFile != nil {
		return fl.currentFile.Close()
	}

	return nil
}

// Shutdown 关闭日志系统
func Shutdown() error {
	// 停止异步日志处理器
	if asyncEnabled {
		close(asyncStopChan)
		asyncWaitGroup.Wait()
		asyncEnabled = false
	}

	fileMux.Lock()
	defer fileMux.Unlock()

	if globalFileLogger != nil {
		return globalFileLogger.close()
	}

	return nil
}
