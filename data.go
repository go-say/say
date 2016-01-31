package say

import "fmt"

// Data is a list of key-value pairs associated with a message.
type Data []KVPair

// KVPair represents a key-value pair.
type KVPair struct {
	Key   string
	Value interface{}
}

// SetData sets a key-value pair that will be printed along with all messages
// sent with this Logger.
func (l *Logger) SetData(data ...interface{}) {
	mu.Lock()
	l.data = l.data[:0]
	err := l.data.appendData(data)
	mu.Unlock()
	if err != nil {
		panic(err)
	}
}

// SetData sets a key-value pair that will be printed along with all messages
// sent with the package-level functions.
func SetData(data ...interface{}) {
	defaultLogger.SetData(data...)
}

// AddData adds a key-value pair that will be printed along with all messages
// sent with this Logger.
func (l *Logger) AddData(key string, value interface{}) {
	if err := isKeyValid(key); err != nil {
		panic(err)
	}

	mu.Lock()
	l.data = append(l.data, KVPair{Key: key, Value: filterDataValue(value)})
	defer mu.Unlock()
}

// AddData adds a key-value pair that will be printed along with all messages
// sent with the package-level functions.
func AddData(key string, value interface{}) {
	defaultLogger.AddData(key, value)
}

func (d *Data) appendData(data []interface{}) error {
	if len(data)%2 != 0 {
		return errOddNumArgs
	}

	for i := 0; i < len(data)/2; i++ {
		key, ok := data[2*i].(string)
		if !ok {
			return errKeyNotString
		}
		if err := isKeyValid(key); err != nil {
			return err
		}
		*d = append(*d, KVPair{
			Key:   key,
			Value: filterDataValue(data[2*i+1]),
		})
	}
	return nil
}

func filterDataValue(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return t
	case error:
		return t.Error()
	case fmt.Stringer:
		return t.String()
	case func() string:
		return t()
	case Hook:
		return t
	case int:
		return t
	case uint:
		return t
	case int64:
		return t
	case uint64:
		return t
	case int32:
		return t
	case uint32:
		return t
	case int16:
		return t
	case uint16:
		return t
	case int8:
		return t
	case uint8:
		return t
	case bool:
		return t
	case float64:
		return t
	case float32:
		return t
	default:
		buf := getBuffer()
		buf.appendInterface(v)
		return buf.String()
	}
}

// Get gets the value associated with the given key as an unquoted string.
// If the given key does not exists ok is false.
func (d Data) Get(key string) (value interface{}, ok bool) {
	if d == nil {
		return nil, false
	}

	for _, v := range d {
		if v.Key == key {
			value = v.Value
			ok = true
		}
	}
	return value, ok
}
