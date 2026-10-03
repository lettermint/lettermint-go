package lettermint

import (
	"context"
	"iter"
)

// WebhooksService manages webhook endpoints. Needs a team token. To verify
// incoming deliveries, use NewWebhook.
type WebhooksService struct {
	service
	// Deliveries reads the delivery attempts of a webhook.
	Deliveries *WebhookDeliveriesService
}

// List returns one page of webhooks. Pass nil for no query parameters.
func (s *WebhooksService) List(ctx context.Context, query *ListWebhooksQuery, options ...RequestOption) (*ListWebhooksResponse, error) {
	return callJSON[ListWebhooksResponse](ctx, s.t, opListWebhooks, callArgs{label: "Webhooks.List", query: query, options: requestOptions(options)})
}

// Iterate yields every webhook, following next_cursor.
func (s *WebhooksService) Iterate(ctx context.Context, query *ListWebhooksQuery, options ...RequestOption) iter.Seq2[WebhookListData, error] {
	return iterate[WebhookListData](ctx, s.t, opListWebhooks, callArgs{label: "Webhooks.Iterate", query: query, options: requestOptions(options)})
}

// Create creates a webhook. The response holds its signing secret once.
func (s *WebhooksService) Create(ctx context.Context, body StoreWebhookData, options ...RequestOption) (*WebhookSecretResponse, error) {
	return callJSON[WebhookSecretResponse](ctx, s.t, opCreateWebhook, callArgs{label: "Webhooks.Create", body: body, options: requestOptions(options)})
}

// Retrieve returns a webhook.
func (s *WebhooksService) Retrieve(ctx context.Context, webhookID string, options ...RequestOption) (*WebhookData, error) {
	return callJSON[WebhookData](ctx, s.t, opGetWebhook, callArgs{label: "Webhooks.Retrieve", path: []string{webhookID}, options: requestOptions(options)})
}

// Update changes a webhook. Fields left absent keep their value; set
// BasicAuth to lettermint.Null[lettermint.WebhookBasicAuthData]() to remove
// the credentials.
func (s *WebhooksService) Update(ctx context.Context, webhookID string, body UpdateWebhookData, options ...RequestOption) (*WebhookMutationResponse, error) {
	return callJSON[WebhookMutationResponse](ctx, s.t, opUpdateWebhook, callArgs{label: "Webhooks.Update", path: []string{webhookID}, body: body, options: requestOptions(options)})
}

// Delete removes a webhook.
func (s *WebhooksService) Delete(ctx context.Context, webhookID string, options ...RequestOption) (*MessageResponse, error) {
	return callJSON[MessageResponse](ctx, s.t, opDeleteWebhook, callArgs{label: "Webhooks.Delete", path: []string{webhookID}, options: requestOptions(options)})
}

// Test sends a webhook.test delivery.
func (s *WebhooksService) Test(ctx context.Context, webhookID string, options ...RequestOption) (*TestWebhookResponse, error) {
	return callJSON[TestWebhookResponse](ctx, s.t, opTestWebhook, callArgs{label: "Webhooks.Test", path: []string{webhookID}, options: requestOptions(options)})
}

// RegenerateSecret replaces the signing secret. The response holds the new
// secret once.
func (s *WebhooksService) RegenerateSecret(ctx context.Context, webhookID string, options ...RequestOption) (*WebhookSecretResponse, error) {
	return callJSON[WebhookSecretResponse](ctx, s.t, opRegenerateWebhookSecret, callArgs{label: "Webhooks.RegenerateSecret", path: []string{webhookID}, options: requestOptions(options)})
}

// WebhookDeliveriesService reads the delivery attempts of a webhook. Needs a
// team token.
type WebhookDeliveriesService struct{ service }

// List returns one page of the deliveries of a webhook. Pass nil for no
// query parameters.
func (s *WebhookDeliveriesService) List(ctx context.Context, webhookID string, query *ListWebhookDeliveriesQuery, options ...RequestOption) (*ListWebhookDeliveriesResponse, error) {
	return callJSON[ListWebhookDeliveriesResponse](ctx, s.t, opListWebhookDeliveries, callArgs{label: "Webhooks.Deliveries.List", path: []string{webhookID}, query: query, options: requestOptions(options)})
}

// Iterate yields every delivery of a webhook, following next_cursor.
func (s *WebhookDeliveriesService) Iterate(ctx context.Context, webhookID string, query *ListWebhookDeliveriesQuery, options ...RequestOption) iter.Seq2[WebhookDeliveryListData, error] {
	return iterate[WebhookDeliveryListData](ctx, s.t, opListWebhookDeliveries, callArgs{label: "Webhooks.Deliveries.Iterate", path: []string{webhookID}, query: query, options: requestOptions(options)})
}

// Retrieve returns one delivery of a webhook.
func (s *WebhookDeliveriesService) Retrieve(ctx context.Context, webhookID, deliveryID string, options ...RequestOption) (*WebhookDeliveryData, error) {
	return callJSON[WebhookDeliveryData](ctx, s.t, opGetWebhookDelivery, callArgs{label: "Webhooks.Deliveries.Retrieve", path: []string{webhookID, deliveryID}, options: requestOptions(options)})
}
