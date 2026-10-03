package lettermint

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// queryPair is one encoded query parameter, in the order it is sent.
type queryPair struct {
	key   string
	value string
}

// encodeQuery serializes a generated query struct to the API's bracket
// syntax, exactly like the Node SDK:
//
//   - the wire name comes from the query tag (page[size], filter[status]);
//     nested structs add brackets: filter[tags][0][name];
//   - booleans are 1 or 0;
//   - slices of values are comma-separated: sort=-created_at,domain;
//   - slices of structs are indexed: filter[tags][0][name]=a&filter[tags][1][name]=b;
//   - nil pointers, empty strings, zero numbers and empty slices are not sent.
func encodeQuery(query any) ([]queryPair, error) {
	if query == nil {
		return nil, nil
	}
	value := reflect.ValueOf(query)
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("query parameters must be a query struct, not %s", value.Type())
	}
	var pairs []queryPair
	appendStruct(&pairs, "", value)
	return pairs, nil
}

func appendStruct(pairs *[]queryPair, prefix string, value reflect.Value) {
	typ := value.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		name := field.Tag.Get("query")
		if name == "" {
			name, _, _ = strings.Cut(field.Tag.Get("json"), ",")
		}
		if name == "" || name == "-" {
			continue
		}
		key := name
		if prefix != "" {
			key = prefix + "[" + name + "]"
		}
		appendValue(pairs, key, value.Field(i))
	}
}

func appendValue(pairs *[]queryPair, key string, value reflect.Value) {
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !value.IsNil() {
			appendValue(pairs, key, value.Elem())
		}
	case reflect.Slice, reflect.Array:
		if value.Len() == 0 {
			return
		}
		if isScalarList(value) {
			items := make([]string, 0, value.Len())
			for i := range value.Len() {
				if text, ok := scalar(value.Index(i)); ok {
					items = append(items, text)
				}
			}
			if len(items) > 0 {
				*pairs = append(*pairs, queryPair{key, strings.Join(items, ",")})
			}
			return
		}
		for i := range value.Len() {
			appendValue(pairs, key+"["+strconv.Itoa(i)+"]", value.Index(i))
		}
	case reflect.Map:
		keys := value.MapKeys()
		sort.Slice(keys, func(a, b int) bool { return fmt.Sprint(keys[a]) < fmt.Sprint(keys[b]) })
		for _, name := range keys {
			appendValue(pairs, key+"["+fmt.Sprint(name)+"]", value.MapIndex(name))
		}
	case reflect.Struct:
		if text, ok := scalar(value); ok {
			*pairs = append(*pairs, queryPair{key, text})
			return
		}
		appendStruct(pairs, key, value)
	default:
		if value.IsZero() && value.Kind() != reflect.Bool {
			return
		}
		if text, ok := scalar(value); ok {
			*pairs = append(*pairs, queryPair{key, text})
		}
	}
}

func isScalarList(value reflect.Value) bool {
	elem := value.Type().Elem()
	for elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}
	switch elem.Kind() {
	case reflect.Struct:
		return elem == reflect.TypeFor[time.Time]()
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Interface:
		return false
	}
	return true
}

// scalar formats one value: booleans as 1/0, times as ISO 8601 in UTC.
func scalar(value reflect.Value) (string, bool) {
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return "", false
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return "1", true
		}
		return "0", true
	case reflect.String:
		return value.String(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(value.Uint(), 10), true
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64), true
	case reflect.Struct:
		if t, ok := value.Interface().(time.Time); ok {
			return t.UTC().Format("2006-01-02T15:04:05.000Z07:00"), true
		}
	}
	return "", false
}

// setQueryParam sets one parameter, replacing it in place when it is already
// there, otherwise appending it.
func setQueryParam(pairs []queryPair, key, value string) []queryPair {
	result := make([]queryPair, 0, len(pairs)+1)
	found := false
	for _, pair := range pairs {
		if pair.key == key {
			if !found {
				result = append(result, queryPair{key, value})
				found = true
			}
			continue
		}
		result = append(result, pair)
	}
	if !found {
		result = append(result, queryPair{key, value})
	}
	return result
}

// formEncode encodes the pairs as application/x-www-form-urlencoded, byte for
// byte like the WHATWG URLSearchParams serializer the Node SDK uses: letters,
// digits and *-._ stay, a space becomes +, and everything else is
// percent-encoded.
func formEncode(pairs []queryPair) string {
	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts = append(parts, formEscape(pair.key)+"="+formEscape(pair.value))
	}
	return strings.Join(parts, "&")
}

func formEscape(value string) string {
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case isAlphanumeric(c) || c == '*' || c == '-' || c == '.' || c == '_':
			builder.WriteByte(c)
		case c == ' ':
			builder.WriteByte('+')
		default:
			fmt.Fprintf(&builder, "%%%02X", c)
		}
	}
	return builder.String()
}
