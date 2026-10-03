package lettermint

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

const (
	redacted = "[redacted]"
	notSet   = "<not set>"
)

// secret holds a token or signing secret. Every way of printing, logging or
// encoding it shows "[redacted]"; only reveal returns the value.
type secret string

func (secret) String() string               { return redacted }
func (secret) GoString() string             { return redacted }
func (secret) Format(f fmt.State, _ rune)   { _, _ = f.Write([]byte(redacted)) }
func (secret) LogValue() slog.Value         { return slog.StringValue(redacted) }
func (secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }
func (secret) MarshalText() ([]byte, error) { return []byte(redacted), nil }

func (s secret) reveal() string { return string(s) }

// shown is what a view displays for the secret: [redacted], or <not set>.
func (s secret) shown() string {
	if s == "" {
		return notSet
	}
	return redacted
}

// view is the printable state of an SDK object: field names and values that
// are safe to log. Objects that hold credentials render a view instead of
// their fields.
type view struct {
	name   string
	fields []viewField
}

type viewField struct {
	name  string
	value any
}

func (v view) String() string {
	parts := make([]string, 0, len(v.fields))
	for _, field := range v.fields {
		parts = append(parts, fmt.Sprintf("%s: %v", field.name, field.value))
	}
	return "lettermint." + v.name + "{" + strings.Join(parts, ", ") + "}"
}

// format implements fmt.Formatter for every verb, so that %v, %+v, %#v, %s
// and %q never print the underlying struct fields.
func (v view) format(f fmt.State, verb rune) {
	text := v.String()
	if verb == 'q' {
		text = fmt.Sprintf("%q", text)
	}
	_, _ = f.Write([]byte(text))
}

func (v view) logValue() slog.Value {
	attrs := make([]slog.Attr, 0, len(v.fields))
	for _, field := range v.fields {
		if value, ok := field.value.(slog.Value); ok {
			attrs = append(attrs, slog.Attr{Key: field.name, Value: value})
		} else {
			attrs = append(attrs, slog.Any(field.name, field.value))
		}
	}
	return slog.GroupValue(attrs...)
}

func (v view) marshalJSON() ([]byte, error) {
	object := make(map[string]any, len(v.fields))
	for _, field := range v.fields {
		if field.value == notSet {
			continue
		}
		if value, ok := field.value.(slog.Value); ok {
			object[field.name] = value.Any()
		} else {
			object[field.name] = field.value
		}
	}
	return json.Marshal(object)
}

// service is embedded in every sub-client. It holds the transport, and with
// it the tokens, in an unexported field; printing, logging or encoding a
// sub-client shows only its name.
type service struct {
	t    *transport
	name string
}

func (s service) view() view { return view{name: s.name} }

// String returns the sub-client's name. It never contains a token.
func (s service) String() string { return s.view().String() }

// GoString returns the sub-client's name. It never contains a token.
func (s service) GoString() string { return s.view().String() }

// Format prints the sub-client's name for every verb. It never prints a token.
func (s service) Format(f fmt.State, verb rune) { s.view().format(f, verb) }

// LogValue implements slog.LogValuer without credentials.
func (s service) LogValue() slog.Value { return s.view().logValue() }

// MarshalJSON encodes an empty object: a sub-client holds no data to share.
func (s service) MarshalJSON() ([]byte, error) { return []byte("{}"), nil }
