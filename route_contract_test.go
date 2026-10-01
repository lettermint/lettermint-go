package lettermint

import (
	"encoding/json"
	"testing"
)

func TestRouteInboundDomainContract(t *testing.T) {
	for _, field := range []string{`,"inbound_route_domain":"incoming.example.com"`, `,"inbound_route_domain":null`, ""} {
		var route RouteData
		body := `{"id":"route_1","project_id":"project_1","slug":"incoming","name":"Incoming","route_type":"inbound","is_default":false,"created_at":"2026-10-01T12:00:00Z","updated_at":"2026-10-01T12:00:00Z"` + field + "}"
		if err := json.Unmarshal([]byte(body), &route); err != nil {
			t.Fatal(err)
		}
		if field == `,"inbound_route_domain":"incoming.example.com"` {
			if route.InboundRouteDomain == nil || *route.InboundRouteDomain != "incoming.example.com" {
				t.Fatalf("domain not decoded: %#v", route)
			}
		} else if route.InboundRouteDomain != nil {
			t.Fatalf("null or absent domain: %#v", route.InboundRouteDomain)
		}
		encoded, err := json.Marshal(route)
		if err != nil {
			t.Fatal(err)
		}
		var decoded RouteData
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		if route.InboundRouteDomain != nil && (decoded.InboundRouteDomain == nil || *decoded.InboundRouteDomain != *route.InboundRouteDomain) {
			t.Fatal("domain did not round trip")
		}
	}
}
