package listen

import "strconv"

// Data is a list of key-value pairs associated with a message.
type Data []*KVPair

// KVPair represents a key-value pair.
type KVPair struct {
	Key   string
	Value string
}

// Data parses and returns the key-value pairs associated with a message.
func (m *Message) Data() Data {
	if len(m.rawData) == 0 {
		return nil
	}

	raw := m.rawData
	data := make(Data, 0, 8)
	for len(raw) > 0 {
		n, key := parseKey(raw)
		if n == 0 {
			m.invalidData()
			return nil
		}
		raw = raw[n:]

		if len(raw) == 0 {
			m.invalidData()
			return nil
		}

		n, val := parseValue(raw)
		if n == 0 {
			m.invalidData()
			return nil
		}
		raw = raw[n:]

		if len(raw) != 0 {
			if raw[0] != ' ' {
				m.invalidData()
				return nil
			}
			raw = raw[1:]
		}
		data = append(data, &KVPair{Key: key, Value: val})
	}
	return data
}

func parseKey(s string) (int, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return i + 1, s[:i]
		}
	}
	return 0, ""
}

func parseValue(s string) (int, string) {
	// Check if it is a quoted value.
	if s[0] == '"' {
		for i := 1; i < len(s); i++ {
			if s[i] == '"' {
				return i + 1, s[:i+1]
			}
		}
		return 0, ""
	}

	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			return i, s[:i]
		}
	}
	return len(s), s
}

func (m *Message) invalidData() {
	errorf("listen: invalid data string: %q", m.rawData)
}

func (d Data) value(key string) string {
	if d == nil {
		return ""
	}

	var value string
	for _, v := range d {
		if v.Key == key {
			value = v.Value
		}
	}
	return value
}

// Has returns whether the given key exists.
func (d Data) Has(key string) bool {
	return d.value(key) != ""
}

// GetString gets the value associated with the given key as an unquoted string.
// If the given key does not exists or cannot be unquoted ok is false.
func (d Data) GetString(key string) (s string, ok bool) {
	if d == nil {
		return
	}

	v := d.value(key)
	if v != "" {
		var err error
		s, err = strconv.Unquote(v)
		ok = err == nil
	}
	return s, ok
}

// GetInt gets the value associated with the given key as an int. If the
// given key does not exists or cannot be converted to an int, ok is false.
func (d Data) GetInt(key string) (i int, ok bool) {
	if d == nil {
		return
	}

	v := d.value(key)
	if v != "" {
		var err error
		i, err = strconv.Atoi(v)
		ok = true
		if err != nil {
			f, err := strconv.ParseFloat(v, 64)
			i = int(f)
			ok = err == nil
		}
	}
	return i, ok
}

// GetFloat64 gets the value associated with the given key as a float64. If the
// given key does not exists or cannot be converted to a float64, ok is false.
func (d Data) GetFloat64(key string) (f float64, ok bool) {
	if d == nil {
		return
	}

	v := d.value(key)
	if v != "" {
		var err error
		f, err = strconv.ParseFloat(v, 64)
		ok = err == nil
	}
	return f, ok
}

// GetBool gets the value associated with the given key as a boolean. If the
// given key does not exists or cannot be converted to a boolean, ok is false.
func (d Data) GetBool(key string) (b bool, ok bool) {
	if d == nil {
		return
	}

	v := d.value(key)
	if v != "" {
		b = v == "true"
		ok = v == "true" || v == "false"
	}
	return b, ok
}
