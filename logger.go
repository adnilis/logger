package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

func logAndLevel(level int, format string, v ...interface{}) string {
	if level < logLevel {
		return ""
	}

	s := fmt.Sprintf(format, v...)
	prefix := levelPrefixes[level]

	var str strings.Builder
	str.Grow(len(s) + len(prefix) + 30)

	if actor == nil {
		str.WriteString("[")
		str.WriteString(time.Now().Format("15:04:05.000"))
		str.WriteString("]")
		str.WriteString(prefix)
		str.WriteString(s)
		str.WriteString(RESET)
		str.WriteString("\n")
		fmt.Fprint(os.Stdout, str.String())
		return ""
	}

	str.WriteString(prefix)
	str.WriteString(s)
	str.WriteString(RESET)
	str.WriteString("\n")

	if level == LogLevelError {
		actor.Error(str.String())
	} else {
		actor.Info(str.String())
	}

	// 写入文件日志（无颜色）
	if globalFileLogger != nil {
		var plainStr strings.Builder
		plainStr.Grow(len(s) + 30)
		plainStr.WriteString("[")
		plainStr.WriteString(time.Now().Format("15:04:05.000"))
		plainStr.WriteString("]")
		plainStr.WriteString("【")
		plainStr.WriteString(LogLevelTags[level])
		plainStr.WriteString("】")
		plainStr.WriteString(s)
		plainStr.WriteString("\n")
		globalFileLogger.write(plainStr.String())
	}

	return s
}

/**
 * 输出详细信息日志
 * @param format 输出格式
 * @param v 输出参数
 */
func Debug(format string, v ...interface{}) {
	if asyncEnabled {
		asyncSendLog(LogLevelVerbose, format, v...)
	} else {
		logAndLevel(LogLevelVerbose, format, v...)
	}
}

/**
 * 输出信息日志
 * @param format 输出格式
 * @param v 输出参数
 */
func Info(format string, v ...interface{}) {
	if asyncEnabled {
		asyncSendLog(LogLevelInfo, format, v...)
	} else {
		logAndLevel(LogLevelInfo, format, v...)
	}
}

/**
 * 输出警告日志
 * @param format 输出格式
 * @param v 输出参数
 */
func Warn(format string, v ...interface{}) {
	if asyncEnabled {
		asyncSendLog(LogLevelWarn, format, v...)
	} else {
		logAndLevel(LogLevelWarn, format, v...)
	}
}

/**
 * 输出错误日志
 * @param format 输出格式
 * @param v 输出参数
 * @returns 返回错误信息
 */
func Error(format string, v ...interface{}) string {
	if asyncEnabled {
		asyncSendLog(LogLevelError, format, v...)
		return fmt.Sprintf(format, v...)
	}
	return logAndLevel(LogLevelError, format, v...)
}

// asyncSendLog 发送异步日志消息
func asyncSendLog(level int, format string, v ...interface{}) {
	msg := &logMessage{
		level:  level,
		format: format,
		args:   v,
	}
	select {
	case asyncChan <- msg:
		// 成功发送到 channel
	default:
		// channel 已满，丢弃日志（避免阻塞）
		// 或者可以选择丢弃最老的消息： <- asyncChan; asyncChan <- msg
	}
}

func Panic(ierr interface{}) (error, bool) {
	if ierr == nil {
		return nil, false
	}

	switch err := ierr.(type) {
	case error:
		{
			var st = func(all bool) string {
				// Reserve 1K buffer at first
				buf := make([]byte, 512)

				for {
					size := runtime.Stack(buf, all)
					// The size of the buffer may be not enough to hold the stacktrace,
					// so double the buffer size
					if size == len(buf) {
						buf = make([]byte, len(buf)<<1)
						continue
					}
					break
				}

				return string(buf)
			}
			Error("panic:%v, %v", err, "\nstack:"+st(false))
			return err, true
		}
	case string:
		{
			var st = func(all bool) string {
				// Reserve 1K buffer at first
				buf := make([]byte, 512)

				for {
					size := runtime.Stack(buf, all)
					// The size of the buffer may be not enough to hold the stacktrace,
					// so double the buffer size
					if size == len(buf) {
						buf = make([]byte, len(buf)<<1)
						continue
					}
					break
				}

				return string(buf)
			}
			Error("panic:%v, err:%s", err, "\nstack:"+st(false))
			return fmt.Errorf("%s", err), true
		}

	}
	return nil, false
}
