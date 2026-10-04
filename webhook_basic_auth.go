package lettermint

import (
	"fmt"
	"log/slog"
)

// The Basic Auth password of a webhook is a credential: printing or logging
// WebhookBasicAuthData shows the username only. JSON encoding is unchanged,
// because the API needs the password.

func (d WebhookBasicAuthData) view() view {
	return view{name: "WebhookBasicAuthData", fields: []viewField{{"Username", d.Username}, {"Password", secret(d.Password).shown()}}}
}

// String shows the username; the password is [redacted].
func (d WebhookBasicAuthData) String() string { return d.view().String() }

// GoString shows the username; the password is [redacted].
func (d WebhookBasicAuthData) GoString() string { return d.view().String() }

// Format shows the username for every verb; the password is [redacted].
func (d WebhookBasicAuthData) Format(f fmt.State, verb rune) { d.view().format(f, verb) }

// LogValue implements slog.LogValuer; the password is [redacted].
func (d WebhookBasicAuthData) LogValue() slog.Value { return d.view().logValue() }
