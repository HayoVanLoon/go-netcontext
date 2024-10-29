package netcontext

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"
)

type ParseFunc func(s string) (any, error)

type StringFunc func(a any) string

// An Entry describes how to handle the serialisation and deserialisation of a
// context value.
type Entry struct {
	ctxKey    any
	stringKey string

	parseValue    ParseFunc
	valueToString StringFunc
}

// CtxKey returns the context key.
func (e Entry) CtxKey() any {
	return e.ctxKey
}

// StringKey returns a string representation of a context key.
func (e Entry) StringKey() string {
	return e.stringKey
}

// Unmarshal unmarshalls a value into 'a'. Returns an error if 'a' is not a
// pointer.
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

// Marshal marshals a value into a string.
func (e Entry) Marshal(a any) string {
	return e.valueToString(a)
}

type Config struct {
	HTTPHeaderPrefix   string
	GrpcMetadataPrefix string
	Entries            []Entry
	NoDeadline         bool
	Log                LogFunc
}

// DefaultHeaderPrefix is the default prefix for HTTP headers and gRPC metadata
// keys.
const DefaultHeaderPrefix = "X-Go-Context-"

var config = Config{
	HTTPHeaderPrefix:   DefaultHeaderPrefix,
	GrpcMetadataPrefix: DefaultHeaderPrefix,
	Log:                log.Printf,
}

// Reset resets the configuration to its default state. It is mainly intended
// for unit tests. Normal code should have no reason to call this function.
func Reset() {
	config = Config{
		HTTPHeaderPrefix:   DefaultHeaderPrefix,
		GrpcMetadataPrefix: DefaultHeaderPrefix,
		Log:                log.Printf,
	}
}

// SetPrefixes sets the same header/metadata prefix for both HTTP/gRPC.
func SetPrefixes(prefix string) {
	config.HTTPHeaderPrefix = prefix
	config.GrpcMetadataPrefix = prefix
}

// HTTPHeaderPrefix returns the prefix for HTTP headers.
func HTTPHeaderPrefix() string {
	return config.HTTPHeaderPrefix
}

// SetHTTPHeaderPrefix sets the prefix for HTTP headers.
func SetHTTPHeaderPrefix(prefix string) {
	config.HTTPHeaderPrefix = prefix
}

// GRPCMetadataPrefix returns the prefix for gRPC metadata keys.
func GRPCMetadataPrefix() string {
	return config.GrpcMetadataPrefix
}

// SetGRPCMetadataPrefix sets the prefix for gRPC metadata keys.
func SetGRPCMetadataPrefix(prefix string) {
	config.GrpcMetadataPrefix = prefix
}

// NoStandardDeadLine will disable propagation of the standard Go context
// deadline. By default, it is enabled.
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

// Log logs a message.
func Log(format string, as ...any) {
	if config.Log == nil {
		return
	}
	config.Log(format, as...)
}

func Entries() []Entry {
	return config.Entries
}

// String adds an Entry for a string context value.
func String(ctxKey any, stringKey string) {
	generic(ctxKey, stringKey, func(s string) (any, error) {
		return s, nil
	})
}

// Int adds an Entry for an int context value.
func Int(ctxKey any, stringKey string) {
	generic(ctxKey, stringKey, func(s string) (any, error) {
		i, err := strconv.Atoi(s)
		return i, err
	})
}

// Int32 adds an Entry for an int32 context value.
func Int32(ctxKey any, stringKey string) {
	generic(ctxKey, stringKey, func(s string) (any, error) {
		i, err := strconv.ParseInt(s, 10, 32)
		return int32(i), err //nolint:gosec
	})
}

// Int64 adds an Entry for an int64 context value.
func Int64(ctxKey any, stringKey string) {
	generic(ctxKey, stringKey, func(s string) (any, error) {
		i, err := strconv.ParseInt(s, 10, 64)
		return int32(i), err
	})
}

func generic(ctxKey any, stringKey string, parse func(s string) (any, error)) {
	Set(ctxKey, stringKey, parse, nil)
}

// TimeFormat used for time.Time context values.
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

// Time adds an Entry for a time.Time context value.
func Time(ctxKey any, stringKey string) {
	set(timeEntry(ctxKey, stringKey))
}

// Set adds an Entry with the given parameters. The parser function is
// required. If the stringer function is not provided, DefaultToString will be
// used.
func Set(ctxKey any, stringKey string, parse ParseFunc, toString StringFunc) {
	if parse == nil {
		panic("parser function cannot be nil")
	}
	if toString == nil {
		toString = DefaultToString
	}
	set(Entry{
		ctxKey:        ctxKey,
		stringKey:     stringKey,
		parseValue:    parse,
		valueToString: toString,
	})
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

// DefaultToString is a convenience wrapper around fmt.Sprintf.
func DefaultToString(a any) string {
	return fmt.Sprintf("%v", a)
}
