package log

import (
	"fmt"
	"os"
	"time"
	"sync"
	"bytes"
)

// Level 日志级别
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// Logger 日志器
type Logger struct {
	level  Level
	output *os.File
	mu     sync.Mutex
}

// NewLogger 创建日志器
func NewLogger(path string) *Logger {
	// 确保目录存在
	if dir := getDir(path); dir != "" {
		os.MkdirAll(dir, 0755)
	}
	
	output, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		output = os.Stderr
	}

	return &Logger{
		level:  LevelInfo,
		output: output,
	}
}

// getDir 获取目录路径
func getDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal 致命错误日志
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
	os.Exit(1)
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	buf := new(bytes.Buffer)
	buf.WriteString(time.Now().Format("2006/01/02 15:04:05"))
	buf.WriteString(" [")
	buf.WriteString(levelToString(level))
	buf.WriteString("] ")
	buf.WriteString(fmt.Sprintf(format, args...))
	buf.WriteString("\n")

	l.output.Write(buf.Bytes())
}

func levelToString(level Level) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}