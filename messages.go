package lettermint

import (
	"context"
	"iter"
)

// MessagesService reads sent and received messages. Needs a team token;
// Reschedule and Cancel also accept the sending token when no team token is
// set.
type MessagesService struct{ service }

// List returns one page of messages. Pass nil for no query parameters.
func (s *MessagesService) List(ctx context.Context, query *ListMessagesQuery, options ...RequestOption) (*ListMessagesResponse, error) {
	return callJSON[ListMessagesResponse](ctx, s.t, opListMessages, callArgs{label: "Messages.List", query: query, options: requestOptions(options)})
}

// Iterate yields every message, following next_cursor.
func (s *MessagesService) Iterate(ctx context.Context, query *ListMessagesQuery, options ...RequestOption) iter.Seq2[MessageListData, error] {
	return iterate[MessageListData](ctx, s.t, opListMessages, callArgs{label: "Messages.Iterate", query: query, options: requestOptions(options)})
}

// Retrieve returns a message.
func (s *MessagesService) Retrieve(ctx context.Context, messageID string, options ...RequestOption) (*MessageData, error) {
	return callJSON[MessageData](ctx, s.t, opGetMessage, callArgs{label: "Messages.Retrieve", path: []string{messageID}, options: requestOptions(options)})
}

// Events returns one page of the events of a message.
func (s *MessagesService) Events(ctx context.Context, messageID string, query *ListMessageEventsQuery, options ...RequestOption) (*ListMessageEventsResponse, error) {
	return callJSON[ListMessageEventsResponse](ctx, s.t, opListMessageEvents, callArgs{label: "Messages.Events", path: []string{messageID}, query: query, options: requestOptions(options)})
}

// IterateEvents yields every event of a message, following next_cursor.
func (s *MessagesService) IterateEvents(ctx context.Context, messageID string, query *ListMessageEventsQuery, options ...RequestOption) iter.Seq2[MessageEventData, error] {
	return iterate[MessageEventData](ctx, s.t, opListMessageEvents, callArgs{label: "Messages.IterateEvents", path: []string{messageID}, query: query, options: requestOptions(options)})
}

// Source returns the raw RFC 822 source of a message.
func (s *MessagesService) Source(ctx context.Context, messageID string, options ...RequestOption) (string, error) {
	return callText(ctx, s.t, opGetMessageSource, callArgs{label: "Messages.Source", path: []string{messageID}, options: requestOptions(options)})
}

// HTML returns the HTML body of a message.
func (s *MessagesService) HTML(ctx context.Context, messageID string, options ...RequestOption) (string, error) {
	return callText(ctx, s.t, opGetMessageHtml, callArgs{label: "Messages.HTML", path: []string{messageID}, options: requestOptions(options)})
}

// Text returns the plain-text body of a message.
func (s *MessagesService) Text(ctx context.Context, messageID string, options ...RequestOption) (string, error) {
	return callText(ctx, s.t, opGetMessageText, callArgs{label: "Messages.Text", path: []string{messageID}, options: requestOptions(options)})
}

// Reschedule moves a scheduled message to another delivery time. It uses the
// team token when it is set, otherwise the sending token.
func (s *MessagesService) Reschedule(ctx context.Context, messageID string, body RescheduleMessageRequest, options ...RequestOption) (*ScheduledMessage, error) {
	return callJSON[ScheduledMessage](ctx, s.t, opRescheduleMessage, callArgs{label: "Messages.Reschedule", path: []string{messageID}, body: body, options: requestOptions(options)})
}

// Cancel cancels a scheduled message. It uses the team token when it is set,
// otherwise the sending token.
func (s *MessagesService) Cancel(ctx context.Context, messageID string, options ...RequestOption) (*ScheduledMessage, error) {
	return callJSON[ScheduledMessage](ctx, s.t, opCancelScheduledMessage, callArgs{label: "Messages.Cancel", path: []string{messageID}, options: requestOptions(options)})
}

// Process releases one quarantined inbound message for webhook delivery. It
// accepts WithIdempotencyKey.
func (s *MessagesService) Process(ctx context.Context, messageID string, options ...IdempotentOption) (*ProcessInboundMessageResponse, error) {
	return callJSON[ProcessInboundMessageResponse](ctx, s.t, opProcessInboundMessage, callArgs{label: "Messages.Process", path: []string{messageID}, options: idempotentOptions(options)})
}
