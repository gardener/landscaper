package buflogr

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/go-logr/logr"
)

const (
	LevelError = "ERROR"
	LevelInfo  = "INFO"
	LevelV     = "V[%d]"
)

var (
	// NameSeparator separates names for logr.WithName.
	NameSeparator = "/"
	// KVFormatter is a function that renders a slice of key/value pairs into a
	// string with the signature: `func(kv ...interface{}) string`.
	KVFormatter = defaultKVFormatter
)

var (
	_ logr.LogSink = &bufLogger{}
	_ Underlier    = &bufLogger{}
)

// New returns a logr.Logger with logr.LogSink implemented by bufLogger using a
// new bytes.Buffer.
func New() logr.Logger {
	return NewWithBuffer(nil)
}

// New returns a logr.Logger with logr.LogSink implemented by bufLogger with an
// existing bytes.Buffer.
func NewWithBuffer(b *bytes.Buffer) logr.Logger {
	if b == nil {
		b = &bytes.Buffer{}
	}
	bl := &bufLogger{
		verbosity: 9,
		buf:       b,
	}
	return logr.New(bl)
}

// bufLogger implements the LogSink interface.
type bufLogger struct {
	name      string
	values    []interface{}
	verbosity int
	buf       *bytes.Buffer // pointer so logged text are persisted across all invocations
	mu        sync.Mutex
}

// Init is not implemented and does not use any runtime info.
func (l *bufLogger) Init(info logr.RuntimeInfo) {
	// not implemented
}

// Enabled implements logr.Logger.Enabled by checking if the current
// verbosity level is less or equal than the logger's maximum verbosity.
func (l *bufLogger) Enabled(level int) bool {
	return level <= l.verbosity
}

// Info implements logr.Logger.Info by writing the line to the internal buffer.
func (l *bufLogger) Info(level int, msg string, kv ...interface{}) {
	if l.Enabled(level) {
		l.writeLine(l.levelString(level), msg, kv...)
	}
}

// Error implements logr.Logger.Error by prefixing the line with "ERROR" and
// write it to the internal buffer.
func (l *bufLogger) Error(err error, msg string, kv ...interface{}) {
	l.writeLine(fmt.Sprintf("%s %v", LevelError, err), msg, kv...)
}

// WithValues returns a new LogSink with additional key/value pairs.
// The new LogSink has a new sync.Mutex.
func (l *bufLogger) WithValues(kv ...interface{}) logr.LogSink {
	return &bufLogger{
		name:      l.name,
		values:    append(l.values, kv...),
		verbosity: l.verbosity,
		buf:       l.buf,
	}
}

// WithName returns a new LogSink with the specified name, appended with
// NameSeparator if there is already a name.
// The new LogSink has a new sync.Mutex.
func (l *bufLogger) WithName(name string) logr.LogSink {
	if l.name != "" {
		name = l.name + NameSeparator + name
	}
	return &bufLogger{
		name:      name,
		values:    l.values,
		verbosity: l.verbosity,
		buf:       l.buf,
	}
}

func defaultKVFormatter(kv ...interface{}) string {
	s := strings.Join(strings.Fields(fmt.Sprint(kv...)), " ")
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	return s
}

func (l *bufLogger) writeLine(level, msg string, kv ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var line []string

	fields := []string{level, l.name, msg, KVFormatter(l.values), KVFormatter(kv)}
	for _, f := range fields {
		if f != "" {
			line = append(line, f)
		}
	}

	l.buf.WriteString(strings.Join(line, " "))
	if !strings.HasSuffix(line[len(line)-1], "\n") {
		l.buf.WriteRune('\n')
	}
}

func (l *bufLogger) levelString(level int) string {
	if level > 0 {
		return fmt.Sprintf(LevelV, level)
	}
	return LevelInfo
}

// Underlier exposes access to the underlying testing.T instance. Since
// callers only have a logr.Logger, they have to know which
// implementation is in use, so this interface is less of an
// abstraction and more of a way to test type conversion.
type Underlier interface {
	GetUnderlying() *bufLogger
}

// GetUnderlying returns the bufLogger underneath this logSink.
func (l *bufLogger) GetUnderlying() *bufLogger {
	return l
}

// Additional methods

// Buf returns the internal buffer.
// Wrap with Mutex().Lock() and Mutex().Unlock() when doing write calls to
// preserve the write order.
func (l *bufLogger) Buf() *bytes.Buffer {
	return l.buf
}

// Mutex returns the sync.Mutex used to preserve the order of writes to the buffer.
func (l *bufLogger) Mutex() *sync.Mutex {
	return &l.mu
}

// Reset clears the buffer, name, and values, as well as resets the verbosity to
// maximum (9).
func (l *bufLogger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.name = ""
	l.values = nil
	l.verbosity = 9
	l.buf = &bytes.Buffer{}
}
