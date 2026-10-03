package lettermint_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	lettermint "github.com/lettermint/lettermint-go/v3"
)

// fakeAPI answers like the Lettermint API, so the examples run without a network.
func fakeAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/send":
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"message_id":"msg_123","status":"pending"}`)
		case "/v1/domains":
			if r.URL.Query().Get("page[cursor]") == "" {
				fmt.Fprint(w, `{"data":[{"id":"d1","domain":"acme.com","status":"verified"}],"next_cursor":"c2"}`)
			} else {
				fmt.Fprint(w, `{"data":[{"id":"d2","domain":"acme.org","status":"verified"}],"next_cursor":null}`)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"Not Found"}`)
		}
	}))
}

func ExampleNew() {
	client, err := lettermint.New(
		lettermint.WithSendingToken(os.Getenv("LETTERMINT_PROJECT_TOKEN")),
		lettermint.WithTeamToken(os.Getenv("LETTERMINT_TEAM_TOKEN")),
		lettermint.WithTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleEmailsService_Compose() {
	api := fakeAPI()
	defer api.Close()
	client, _ := lettermint.New(lettermint.WithSendingToken("lm_example"), lettermint.WithBaseURL(api.URL+"/v1"))

	welcome := client.Emails.Compose().From("Acme <hello@acme.com>").Subject("Welcome to Acme")
	resp, err := welcome.To("jane@example.com").HTML("<p>Hi Jane</p>").Send(context.Background(), lettermint.WithIdempotencyKey("welcome-jane"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.MessageID, resp.Status)
	// Output: msg_123 pending
}

func ExampleEmailsService_Send() {
	api := fakeAPI()
	defer api.Close()
	client, _ := lettermint.New(lettermint.WithSendingToken("lm_example"), lettermint.WithBaseURL(api.URL+"/v1"))

	resp, err := client.Emails.Send(context.Background(), lettermint.SendMailRequest{
		From:    "Acme <hello@acme.com>",
		To:      []string{"jane@example.com"},
		Subject: "Your order has shipped",
		Text:    lettermint.Value("It is on its way."),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Status)
	// Output: pending
}

func ExampleDomainsService_Iterate() {
	api := fakeAPI()
	defer api.Close()
	client, _ := lettermint.New(lettermint.WithTeamToken("lm_team_example"), lettermint.WithBaseURL(api.URL+"/v1"))

	for domain, err := range client.Domains.Iterate(context.Background(), &lettermint.ListDomainsQuery{FilterStatus: lettermint.DomainStatusVerified}) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(domain.Domain)
	}
	// Output:
	// acme.com
	// acme.org
}

func ExampleAPIError() {
	api := fakeAPI()
	defer api.Close()
	client, _ := lettermint.New(lettermint.WithTeamToken("lm_team_example"), lettermint.WithBaseURL(api.URL+"/v1"))

	_, err := client.Projects.Retrieve(context.Background(), "missing", nil)
	var notFound *lettermint.NotFoundError
	var apiErr *lettermint.APIError
	fmt.Println(errors.As(err, &notFound), errors.As(err, &apiErr), apiErr.Status)
	// Output: true true 404
}

func ExampleWebhook_VerifyRequest() {
	webhook, err := lettermint.NewWebhook(os.Getenv("LETTERMINT_WEBHOOK_SECRET"))
	if err != nil {
		return
	}
	http.HandleFunc("POST /webhooks/lettermint", func(w http.ResponseWriter, r *http.Request) {
		event, err := webhook.VerifyRequest(r)
		if err != nil {
			http.Error(w, "invalid signature", http.StatusBadRequest)
			return
		}
		log.Println(event.Event)
		w.WriteHeader(http.StatusNoContent)
	})
}

func ExampleNewFromToken() {
	_, err := lettermint.NewFromToken("lm_sso_abc123")
	var configErr *lettermint.ConfigError
	fmt.Println(errors.As(err, &configErr))
	// Output: true
}
