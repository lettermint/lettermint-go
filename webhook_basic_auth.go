package lettermint

func SetWebhookBasicAuth(username, password string) **WebhookBasicAuthData {
	credentials := &WebhookBasicAuthData{Username: username, Password: password}
	return &credentials
}

func ClearWebhookBasicAuth() **WebhookBasicAuthData {
	var credentials *WebhookBasicAuthData
	return &credentials
}

func (WebhookBasicAuthData) String() string {
	return "WebhookBasicAuthData{<redacted>}"
}

func (credentials WebhookBasicAuthData) GoString() string {
	return credentials.String()
}
