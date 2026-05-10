# Upgrade To v2

This guide covers upgrading from the latest released v1 Go SDK to v2.

## Highlights

- Sending email continues to use `lettermint.New(token)`.
- The full Lettermint API is available through `lettermint.NewAPI(token)`.
- Sending tokens use `x-lettermint-token`; full API tokens use `Authorization: Bearer`.
- `Ping` returns the raw trimmed `pong` response.
- API request and response structs are generated from the OpenAPI specs.

## Sending

```go
client, err := lettermint.New("sending-token")
if err != nil {
    return err
}

pong, err := client.Ping(ctx)
```

## Full API

```go
api, err := lettermint.NewAPI("api-token")
if err != nil {
    return err
}

domains, err := api.Domains.List(ctx, nil)
```

## Batch Sending

```go
_, err = client.SendBatch(ctx, []lettermint.SendMailRequest{{
    From: "sender@example.com",
    To: []string{"user@example.com"},
    Subject: "Hello",
    Text: "Hi",
}})
```

