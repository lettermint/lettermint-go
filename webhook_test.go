package lettermint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const webhookSecret = "whsec_testSecretValue0123456789abcdef"

var webhookNow = time.Unix(1767225570, 0)

const webhookBody = `{"id":"d1","event":"message.delivered","timestamp":"2026-01-01T00:00:00Z","data":{"message_id":"m1","recipient":"jane@example.test"}}`

func sign(secret string, t int64, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(t, 10) + "." + body))
	return hex.EncodeToString(mac.Sum(nil))
}

func signedHeaders(t int64, body string) http.Header {
	return http.Header{
		HeaderSignature: {fmt.Sprintf("t=%d,v1=%s", t, sign(webhookSecret, t, body))},
		HeaderDelivery:  {strconv.FormatInt(t, 10)},
	}
}

func testWebhook(t *testing.T, options ...WebhookOption) *Webhook {
	t.Helper()
	w, err := NewWebhook(webhookSecret, append([]WebhookOption{WithClock(func() time.Time { return webhookNow })}, options...)...)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func reason(t *testing.T, err error) WebhookVerificationReason {
	t.Helper()
	return mustAs[*WebhookVerificationError](t, err).Reason
}

func TestWebhookVerifiesAndReturnsThePayload(t *testing.T) {
	w := testWebhook(t)
	payload, err := w.Verify([]byte(webhookBody), signedHeaders(webhookNow.Unix(), webhookBody))
	if err != nil {
		t.Fatal(err)
	}
	if payload.ID != "d1" || payload.Event != WebhookEventMessageDelivered || payload.Timestamp != "2026-01-01T00:00:00Z" || string(payload.Raw) != webhookBody {
		t.Fatalf("%+v", payload)
	}
	var data struct {
		MessageID string `json:"message_id"`
	}
	if err := payload.DecodeData(&data); err != nil || data.MessageID != "m1" {
		t.Fatal(data, err)
	}
}

func TestWebhookTolerance(t *testing.T) {
	w := testWebhook(t)
	for _, offset := range []int64{-300, 0, 300} {
		at := webhookNow.Unix() + offset
		if _, err := w.Verify([]byte(webhookBody), signedHeaders(at, webhookBody)); err != nil {
			t.Errorf("%d: %v", offset, err)
		}
	}
	for _, offset := range []int64{-301, 301} {
		at := webhookNow.Unix() + offset
		_, err := w.Verify([]byte(webhookBody), signedHeaders(at, webhookBody))
		if reason(t, err) != WebhookTimestampOutOfTolerance {
			t.Error(offset, err)
		}
	}
	strict := testWebhook(t, WithTolerance(0))
	if _, err := strict.Verify([]byte(webhookBody), signedHeaders(webhookNow.Unix(), webhookBody)); err != nil {
		t.Fatal(err)
	}
	_, err := strict.Verify([]byte(webhookBody), signedHeaders(webhookNow.Unix()-1, webhookBody))
	if reason(t, err) != WebhookTimestampOutOfTolerance {
		t.Fatal("zero keeps the check enabled")
	}
	if _, err := NewWebhook(webhookSecret, WithTolerance(-time.Second)); err == nil {
		t.Fatal("negative tolerance")
	}
	_, err = NewWebhook("")
	mustAs[*ConfigError](t, err)
}

func TestWebhookRejections(t *testing.T) {
	w := testWebhook(t)
	at := webhookNow.Unix()
	good := sign(webhookSecret, at, webhookBody)
	cases := map[string]struct {
		body    string
		headers http.Header
		want    WebhookVerificationReason
	}{
		"changed body":       {webhookBody + " ", signedHeaders(at, webhookBody), WebhookSignatureMismatch},
		"wrong secret":       {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,v1=%s", at, sign("whsec_other", at, webhookBody))}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureMismatch},
		"prefix stripped":    {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,v1=%s", at, sign(strings.TrimPrefix(webhookSecret, "whsec_"), at, webhookBody))}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureMismatch},
		"no signature":       {webhookBody, http.Header{HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMissing},
		"empty signature":    {webhookBody, http.Header{HeaderSignature: {" "}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMissing},
		"two signatures":     {webhookBody, http.Header{HeaderSignature: {"t=1,v1=" + good, "t=1,v1=" + good}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMalformed},
		"no delivery":        {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,v1=%s", at, good)}}, WebhookDeliveryHeaderMissing},
		"delivery mismatch":  {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,v1=%s", at, good)}, HeaderDelivery: {fmt.Sprint(at + 1)}}, WebhookDeliveryTimestampMismatch},
		"empty delivery":     {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,v1=%s", at, good)}, HeaderDelivery: {""}}, WebhookDeliveryTimestampMismatch},
		"two deliveries":     {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,v1=%s", at, good)}, HeaderDelivery: {fmt.Sprint(at), fmt.Sprint(at)}}, WebhookDeliveryTimestampMismatch},
		"duplicate t":        {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,t=%d,v1=%s", at, at, good)}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMalformed},
		"missing t":          {webhookBody, http.Header{HeaderSignature: {"v1=" + good}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMalformed},
		"missing v1":         {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d", at)}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMalformed},
		"short v1":           {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,v1=abc", at)}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMalformed},
		"non-numeric t":      {webhookBody, http.Header{HeaderSignature: {"t=abc,v1=" + good}, HeaderDelivery: {"abc"}}, WebhookSignatureHeaderMalformed},
		"negative t":         {webhookBody, http.Header{HeaderSignature: {"t=-1,v1=" + good}, HeaderDelivery: {"-1"}}, WebhookSignatureHeaderMalformed},
		"huge t":             {webhookBody, http.Header{HeaderSignature: {"t=99999999999999999999,v1=" + good}, HeaderDelivery: {"1"}}, WebhookSignatureHeaderMalformed},
		"non-ASCII":          {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,v1=%sé", at, good[:63])}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMalformed},
		"full-width digits":  {webhookBody, http.Header{HeaderSignature: {"t=１７６７２２５５７０,v1=" + good}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMalformed},
		"control characters": {webhookBody, http.Header{HeaderSignature: {fmt.Sprintf("t=%d,\x00v1=%s", at, good)}, HeaderDelivery: {fmt.Sprint(at)}}, WebhookSignatureHeaderMalformed},
		"empty body":         {"", signedHeaders(at, ""), WebhookBodyInvalid},
		"not JSON":           {"not json", signedHeaders(at, "not json"), WebhookPayloadInvalid},
		"JSON array":         {"[1]", signedHeaders(at, "[1]"), WebhookPayloadInvalid},
		"wrong field types":  {`{"id":1}`, signedHeaders(at, `{"id":1}`), WebhookPayloadInvalid},
	}
	for label, c := range cases {
		_, err := w.Verify([]byte(c.body), c.headers)
		if got := reason(t, err); got != c.want {
			t.Errorf("%s: %s, want %s (%v)", label, got, c.want, err)
		}
	}
}

func TestWebhookSignatureVariants(t *testing.T) {
	w := testWebhook(t)
	at := webhookNow.Unix()
	good := sign(webhookSecret, at, webhookBody)
	other := strings.Repeat("0", 64)
	for _, header := range []string{
		fmt.Sprintf("t=%d,v1=%s,v1=%s", at, good, other),
		fmt.Sprintf("t=%d,v1=%s,v1=%s", at, other, good),
		fmt.Sprintf("t=%d,v1=%s,v2=%s", at, good, other),
		fmt.Sprintf(" t=%d , v1=%s ", at, strings.ToUpper(good)),
		fmt.Sprintf("v1=%s,t=%d,junk", good, at),
	} {
		if _, err := w.VerifySignature([]byte(webhookBody), header, fmt.Sprint(at)); err != nil {
			t.Errorf("%s: %v", header, err)
		}
	}
	// VerifySignature checks the delivery timestamp only when it is given.
	header := fmt.Sprintf("t=%d,v1=%s", at, good)
	if _, err := w.VerifySignature([]byte(webhookBody), header, ""); err != nil {
		t.Fatal(err)
	}
	_, err := w.VerifySignature([]byte(webhookBody), header, fmt.Sprint(at+1))
	if reason(t, err) != WebhookDeliveryTimestampMismatch {
		t.Fatal(err)
	}
}

func TestWebhookReadsHeadersCaseInsensitively(t *testing.T) {
	w := testWebhook(t)
	at := webhookNow.Unix()
	headers := http.Header{
		"x-lettermint-signature": {fmt.Sprintf("t=%d,v1=%s", at, sign(webhookSecret, at, webhookBody))},
		"X-LETTERMINT-DELIVERY":  {fmt.Sprint(at)},
	}
	if _, err := w.Verify([]byte(webhookBody), headers); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookVerifyRequest(t *testing.T) {
	w := testWebhook(t)
	request := httptest.NewRequest("POST", "/webhooks/lettermint", strings.NewReader(webhookBody))
	for key, values := range signedHeaders(webhookNow.Unix(), webhookBody) {
		request.Header[key] = values
	}
	if _, err := w.VerifyRequest(request); err != nil {
		t.Fatal(err)
	}
	again := new(strings.Builder)
	if _, err := fmt.Fprint(again, readAll(t, request)); err != nil || again.String() != webhookBody {
		t.Fatal("the body can be read again")
	}
	small := testWebhook(t, WithMaxBodyBytes(10))
	request = httptest.NewRequest("POST", "/", strings.NewReader(webhookBody))
	_, err := small.VerifyRequest(request)
	if reason(t, err) != WebhookBodyInvalid {
		t.Fatal(err)
	}
}

func readAll(t *testing.T, r *http.Request) string {
	t.Helper()
	var builder strings.Builder
	buffer := make([]byte, 512)
	for {
		n, err := r.Body.Read(buffer)
		builder.Write(buffer[:n])
		if err != nil {
			return builder.String()
		}
	}
}

func TestWebhookNeverShowsTheSecret(t *testing.T) {
	w := testWebhook(t)
	assertNoSecret(t, "webhook", w, webhookSecret)
	assertNoSecret(t, "webhook value", *w, webhookSecret)
	_, err := w.Verify([]byte("{}"), http.Header{HeaderSignature: {"t=1,v1=" + strings.Repeat("0", 64)}, HeaderDelivery: {"1"}})
	assertNoSecret(t, "error", err, webhookSecret)
	if fmt.Sprint(w) != "lettermint.Webhook{Tolerance: 5m0s, Secret: [redacted]}" {
		t.Fatal(fmt.Sprint(w))
	}
}

// The conformance suite's webhook vectors (testdata/webhooks.json, a copy of
// conformance/webhooks.json in lettermint/sdk-generator).
func TestConformanceWebhookVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/webhooks.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		Vectors []struct {
			ID         string            `json:"id"`
			Secret     string            `json:"secret"`
			Headers    map[string]string `json:"headers"`
			BodyBase64 string            `json:"body_base64"`
			Now        int64             `json:"now"`
			Tolerance  int64             `json:"tolerance"`
			Expect     string            `json:"expect"`
			Reason     string            `json:"reason"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(data, &suite); err != nil {
		t.Fatal(err)
	}
	if len(suite.Vectors) != 25 {
		t.Fatalf("%d vectors", len(suite.Vectors))
	}
	for _, vector := range suite.Vectors {
		t.Run(vector.ID, func(t *testing.T) {
			body, _ := base64.StdEncoding.DecodeString(vector.BodyBase64)
			headers := http.Header{}
			for key, value := range vector.Headers {
				headers[key] = []string{value}
			}
			w, err := NewWebhook(vector.Secret, WithTolerance(time.Duration(vector.Tolerance)*time.Second), WithClock(func() time.Time { return time.Unix(vector.Now, 0) }))
			if err != nil {
				t.Fatal(err)
			}
			_, err = w.Verify(body, headers)
			if vector.Expect == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if got := reason(t, err); string(got) != vector.Reason {
				t.Fatalf("%s, want %s", got, vector.Reason)
			}
		})
	}
}
