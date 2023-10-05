// Copyright by sirupsen
//
// file taken from https://github.com/sirupsen/logrus
// add the support for additional fixed fields.
// Because of usage of many unecessarily provide fields,
// types and functions, all the stuff has to be copied
// to be extended.
//

package logrusfmt

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/modern-go/reflect2"
)

const (
	red    = 31
	green  = 32
	yellow = 33
	blue   = 36
	gray   = 37
	dark   = 90
)

var baseTimestamp time.Time

func init() {
	baseTimestamp = time.Now()
}

type FieldFormatters map[string]FieldFormatter

type FieldFormatter func(w io.Writer, key string, value interface{}, needsQuoting func(string) bool)

func PlainValue(w io.Writer, key string, value interface{}, needsQuoting func(string) bool) {
	if reflect2.IsNil(value) {
		w.Write([]byte("<nil>"))
		return
	}
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !needsQuoting(stringVal) {
		w.Write([]byte(stringVal))
	} else {
		w.Write([]byte(fmt.Sprintf("%q", stringVal)))
	}
}

func BracketValue(w io.Writer, key string, value interface{}, needsQuoting func(string) bool) {
	if reflect2.IsNil(value) {
		w.Write([]byte("<nil>"))
		return
	}
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	w.Write([]byte(fmt.Sprintf("[%s]", stringVal)))
}

func KeyValue(w io.Writer, key string, value interface{}, needsQuoting func(string) bool) {
	PlainValue(w, key, key, needsQuoting)
	w.Write([]byte{'='})
	PlainValue(w, key, value, needsQuoting)
}

func LevelValue(w io.Writer, key string, value interface{}, needsQuoting func(string) bool) {
	f := fmt.Sprintf(fmt.Sprintf("%%-%ds", maxlevellength), value.(string))
	PlainValue(w, key, f, func(string) bool { return false })
}

// TextFormatter formats logs into text
type TextFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors.
	DisableColors bool

	// Force quoting of all values
	ForceQuote bool

	// DisableQuote disables quoting for all values.
	// DisableQuote will have a lower priority than ForceQuote.
	// If both of them are set to true, quote will be forced on all values.
	DisableQuote bool

	// Override coloring based on CLICOLOR and CLICOLOR_FORCE. - https://bixense.com/clicolors/
	EnvironmentOverrideColors bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// TimestampFormat to use for display when a full timestamp is printed.
	// The format to use is the same than for time.Format or time.Parse from the standard
	// library.
	// The standard Library already provides a set of predefined format.
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool

	// The keys sorting function, when uninitialized it uses sort.Strings.
	SortingFunc func([]string)

	// Disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool

	// PadLevelText Adds padding the level text so that all the levels output at the same length
	// PadLevelText is a superset of the DisableLevelTruncation option
	PadLevelText bool

	// QuoteEmptyFields will wrap empty fields in quotes if true
	QuoteEmptyFields bool

	// Whether the logger's out is to a terminal
	isTerminal bool

	// FieldMap allows users to customize the names of keys for default fields.
	// As an example:
	// formatter := &TextFormatter{
	//     FieldMap: FieldMap{
	//         FieldKeyTime:  "@timestamp",
	//         FieldKeyLevel: "@level",
	//         FieldKeyMsg:   "@message"}}
	FieldMap FieldMap

	// FixedFields can be used to definen a fixed order for dedicated fields.
	// They will be rendered before other fields.
	// If defined, the standard field keys should be added, also.
	// The default order is: FieldKeyTime, FieldKeyLevel, FieldKeyMsg,
	// FieldKeyFunc, FieldKeyFile.
	FixedFields []string

	// FieldFormatters are used render a key/value pair.
	// The default formatter is KeyValue.
	FieldFormatters FieldFormatters

	// CallerPrettyfier can be set by the user to modify the content
	// of the function and file keys in the data when ReportCaller is
	// activated. If any of the returned value is the empty string the
	// corresponding key will be removed from fields.
	CallerPrettyfier func(*runtime.Frame) (function string, file string)

	terminalInitOnce sync.Once

	// The max length of the level text, generated dynamically on init
	levelTextMaxLength int
}

func (f *TextFormatter) init(entry *Entry) {
	if entry.Logger != nil {
		f.isTerminal = checkIfTerminal(entry.Logger.Out)
	}
	// Get the max length of the level text
	for _, level := range AllLevels {
		levelTextLength := utf8.RuneCount([]byte(level.String()))
		if levelTextLength > f.levelTextMaxLength {
			f.levelTextMaxLength = levelTextLength
		}
	}
}

func (f *TextFormatter) isColored() bool {
	isColored := f.ForceColors || (f.isTerminal && (runtime.GOOS != "windows"))

	if f.EnvironmentOverrideColors {
		switch force, ok := os.LookupEnv("CLICOLOR_FORCE"); {
		case ok && force != "0":
			isColored = true
		case ok && force == "0", os.Getenv("CLICOLOR") == "0":
			isColored = false
		}
	}

	return isColored && !f.DisableColors
}

// Format renders a single log entry
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	f.terminalInitOnce.Do(func() { f.init(entry) })

	data := make(Fields)
	for k, v := range entry.Data {
		data[k] = v
	}
	prefixFieldClashes(data, f.FieldMap, entry.HasCaller())
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	var funcVal, fileVal string

	if entry.HasCaller() {
		if f.CallerPrettyfier != nil {
			funcVal, fileVal = f.CallerPrettyfier(entry.Caller)
		} else {
			funcVal = entry.Caller.Function
			fileVal = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		}
	}

	fixedKeys := make([]string, 0, len(defaultFixedFields)+len(data))

	fixed := defaultFixedFields
	if f.FixedFields != nil {
		fixed = f.FixedFields
	}

	for _, field := range fixed {
		switch field {
		case FieldKeyTime:
			if !f.DisableTimestamp {
				fixedKeys = append(fixedKeys, field)
			}
		case FieldKeyLevel:
			fixedKeys = append(fixedKeys, field)
		case FieldKeyMsg:
			if entry.Message != "" {
				fixedKeys = append(fixedKeys, field)
			}
		case FieldKeyFunc:
			if funcVal != "" {
				fixedKeys = append(fixedKeys, field)
			}
		case FieldKeyFile:
			if fileVal != "" {
				fixedKeys = append(fixedKeys, field)
			}
		default:
			for i := 0; i < len(keys); i++ {
				if keys[i] == field {
					fixedKeys = append(fixedKeys, field)
					keys = append(keys[:i], keys[i+1:]...)
					i--
				}
			}
		}
	}

	if !f.DisableSorting {
		if f.SortingFunc == nil {
			sort.Strings(keys)
			fixedKeys = append(fixedKeys, keys...)
		} else {
			fixedKeys = append(fixedKeys, keys...)
			f.SortingFunc(fixedKeys)
		}
	} else {
		fixedKeys = append(fixedKeys, keys...)
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	var levelColor int
	if f.isColored() {
		switch entry.Level {
		case DebugLevel, TraceLevel:
			levelColor = gray
		case WarnLevel:
			levelColor = yellow
		case ErrorLevel, FatalLevel, PanicLevel:
			levelColor = red
		case InfoLevel:
			levelColor = green
		default:
			levelColor = dark
		}
	}
	if levelColor > 0 {
		fmt.Fprintf(b, "\x1b[%dm", levelColor)
	}

	for _, key := range fixedKeys {
		var value interface{}
		switch key {
		case FieldKeyTime:
			value = entry.Time.Format(timestampFormat)
		case FieldKeyLevel:
			value = entry.Level.String()
		case FieldKeyMsg:
			value = entry.Message
		case FieldKeyFunc:
			value = funcVal
		case FieldKeyFile:
			value = fileVal
		default:
			value = data[key]
		}

		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		f.appendKeyValue(b, key, value)
	}

	if levelColor > 0 {
		fmt.Fprintf(b, "\x1b[0m")
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *TextFormatter) needsQuoting(text string) bool {
	if f.ForceQuote {
		return true
	}
	if f.QuoteEmptyFields && len(text) == 0 {
		return true
	}
	if f.DisableQuote {
		return false
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == ':' || ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}

func (f *TextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	fmt := KeyValue
	if f.FieldFormatters != nil && f.FieldFormatters[key] != nil {
		fmt = f.FieldFormatters[key]
	}
	fmt(b, key, value, f.needsQuoting)
}

func (f *TextFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	if reflect2.IsNil(value) {
		b.WriteString("<nil>")
		return
	}
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !f.needsQuoting(stringVal) {
		b.WriteString(stringVal)
	} else {
		b.WriteString(fmt.Sprintf("%q", stringVal))
	}
}
