package netcontext

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"
)

type Entry struct {
	ctxKey    any
	stringKey string

	parseValue    func(s string) (any, error)
	valueToString func(a any) string
}

func (e Entry) CtxKey() any {
	return e.ctxKey
}

func (e Entry) StringKey() string {
	return e.stringKey
}

func (e Entry) Unmarshal(s string, a any) error {
	x, err := e.parseValue(s)
	if err != nil {
		return err
	}
	vp := reflect.ValueOf(a)
	if vp.Type().Kind() != reflect.Pointer {
		return fmt.Errorf("a must be a pointer")
	}
	va := vp.Elem()
	vx := reflect.ValueOf(x)
	if !vx.Type().AssignableTo(va.Type()) {
		return fmt.Errorf("cannot assign %T to %T", x, a)
	}
	va.Set(reflect.ValueOf(x))
	return nil
}

func (e Entry) ValueToString(a any) string {
	return e.valueToString(a)
}

type Config struct {
	HTTPHeaderPrefix   string
	GrpcMetadataPrefix string
	Entries            []Entry
	NoDeadline         bool
	Log                LogFunc
}

const DefaultHeaderPrefix = "X-Go-Context-"

var config = Config{
	HTTPHeaderPrefix:   DefaultHeaderPrefix,
	GrpcMetadataPrefix: DefaultHeaderPrefix,
	Log:                log.Printf,
}

// Reset resets the configuration to its default state. Normal code should have
// no need to call this function.
func Reset() {
	config = Config{
		HTTPHeaderPrefix:   DefaultHeaderPrefix,
		GrpcMetadataPrefix: DefaultHeaderPrefix,
		Log:                log.Printf,
	}
}

func Entries() []Entry {
	return config.Entries
}

func SetPrefixes(prefix string) {
	config.HTTPHeaderPrefix = prefix
	config.GrpcMetadataPrefix = prefix
}

func HTTPHeaderPrefix() string {
	return config.HTTPHeaderPrefix
}

func SetHTTPHeaderPrefix(prefix string) {
	config.HTTPHeaderPrefix = prefix
}

func GRPCMetadataPrefix() string {
	return config.GrpcMetadataPrefix
}

func SetGRPCMetadataPrefix(prefix string) {
	config.GrpcMetadataPrefix = prefix
}

func set(e Entry) {
	for i := range config.Entries {
		if config.Entries[i].CtxKey() == e.CtxKey() {
			config.Entries[i] = e
			return
		}
	}
	config.Entries = append(config.Entries, e)
}

// NoStandardDeadLine will disable propagation of the standard Go context
// deadline.
func NoStandardDeadLine() {
	config.NoDeadline = true
}

var deadline = timeEntry(nil, "Deadline")

// Deadline returns the Entry to be used for propagating the standard Go
// context.
func Deadline() (Entry, bool) {
	if config.NoDeadline {
		return Entry{}, false
	}
	return deadline, true
}

type LogFunc func(format string, as ...any)

// SetLogger sets the log function. Setting it to nil will disable logging.
func SetLogger(logger LogFunc) {
	config.Log = logger
}

func Log(format string, as ...any) {
	if config.Log == nil {
		return
	}
	config.Log(format, as...)
}

func String(ctxKey any, stringKey string) {
	generic(ctxKey, stringKey, func(s string) (any, error) {
		return s, nil
	})
}

func Int(ctxKey any, stringKey string) {
	generic(ctxKey, stringKey, func(s string) (any, error) {
		i, err := strconv.Atoi(s)
		return i, err
	})
}

func Int32(ctxKey any, stringKey string) {
	generic(ctxKey, stringKey, func(s string) (any, error) {
		i, err := strconv.ParseInt(s, 10, 32)
		return int32(i), err //nolint:gosec
	})
}

func Int64(ctxKey any, stringKey string) {
	generic(ctxKey, stringKey, func(s string) (any, error) {
		i, err := strconv.ParseInt(s, 10, 64)
		return int32(i), err
	})
}

var TimeFormat = time.RFC3339Nano

func timeEntry(ctxKey any, stringKey string) Entry {
	parse := func(s string) (any, error) {
		return time.Parse(TimeFormat, s)
	}
	toString := func(a any) string {
		t, ok := a.(time.Time)
		if !ok {
			return ""
		}
		return t.Format(TimeFormat)
	}
	return Entry{
		ctxKey:        ctxKey,
		stringKey:     stringKey,
		parseValue:    parse,
		valueToString: toString,
	}
}

func Time(ctxKey any, stringKey string) {
	set(timeEntry(ctxKey, stringKey))
}

func generic(ctxKey any, stringKey string, parse func(s string) (any, error)) {
	set(Entry{
		ctxKey:        ctxKey,
		stringKey:     stringKey,
		parseValue:    parse,
		valueToString: defaultToString,
	})
}

func defaultToString(a any) string {
	return fmt.Sprintf("%v", a)
}
