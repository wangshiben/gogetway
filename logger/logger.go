package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

var LoggerFile *os.File
var Logger = log.New(io.Discard, "", 0)

var (
	loggerMu         sync.RWMutex
	debug            bool
	initialized      bool
	configuredDebug  bool
	loggerFileWriter io.Closer
)

const maxLogFileSize int64 = 100 * 1024 * 1024

type rotatingFileWriter struct {
	mu       sync.Mutex
	dir      string
	nameBase string
	file     *os.File
	path     string
	size     int64
	sequence uint64
}

func newRotatingFileWriter(dir string) (*rotatingFileWriter, error) {
	writer := &rotatingFileWriter{
		dir:      dir,
		nameBase: time.Now().Format("2006_01_02_15_04_05"),
	}
	if err := writer.openNextLocked(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *rotatingFileWriter) nextPathLocked() string {
	name := w.nameBase + ".log"
	if w.sequence > 0 {
		name = fmt.Sprintf("%s_%d.log", w.nameBase, w.sequence)
	}
	return w.dir + string(os.PathSeparator) + name
}

func (w *rotatingFileWriter) openNextLocked() error {
	path := w.nextPathLocked()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	oldFile, oldPath := w.file, w.path
	w.file = file
	w.path = path
	w.size = 0
	w.sequence++
	if oldFile != nil {
		_ = oldFile.Close()
		_ = os.Remove(oldPath)
	}
	return nil
}

func (w *rotatingFileWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, os.ErrClosed
	}
	written := 0
	for len(data) > 0 {
		if w.size >= maxLogFileSize {
			if err := w.openNextLocked(); err != nil {
				return written, err
			}
		}
		remaining := maxLogFileSize - w.size
		chunk := data
		if int64(len(chunk)) > remaining {
			chunk = chunk[:remaining]
		}
		n, err := w.file.Write(chunk)
		w.size += int64(n)
		written += n
		data = data[n:]
		if err != nil {
			return written, err
		}
		if n != len(chunk) {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

func (w *rotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// SetDebug controls whether internal diagnostic logs are enabled. Call it
// before InitLogger, normally during CLI startup.
func SetDebug(enabled bool) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	debug = enabled
	if initialized && configuredDebug != enabled {
		initialized = false
	}
}

// DebugEnabled reports the current debug setting.
func DebugEnabled() bool {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return debug
}

func InitLogger() {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	if initialized && configuredDebug == debug {
		return
	}
	if loggerFileWriter != nil {
		_ = loggerFileWriter.Close()
		loggerFileWriter = nil
	} else if LoggerFile != nil {
		_ = LoggerFile.Close()
	}
	LoggerFile = nil

	if debug {
		output := io.Writer(os.Stderr)
		rotating, err := newRotatingFileWriter(".")
		if err == nil {
			LoggerFile = rotating.file
			loggerFileWriter = rotating
			output = io.MultiWriter(os.Stderr, rotating)
		}
		Logger = log.New(output, "INFO ", log.LstdFlags|log.Lmicroseconds)
	} else {
		Logger = log.New(io.Discard, "", 0)
	}
	configuredDebug = debug
	initialized = true
}
func LogInfo(format string) {
	loggerMu.RLock()
	logger := Logger
	loggerMu.RUnlock()
	if logger != nil {
		logger.Println(format)
	}
}

// LogInfof writes a formatted diagnostic message when debug logging is enabled.
func LogInfof(format string, args ...any) {
	loggerMu.RLock()
	logger := Logger
	loggerMu.RUnlock()
	if logger != nil {
		logger.Printf(format, args...)
	}
}
