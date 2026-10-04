// Example: the Team API: list domains page by page and read statistics.
//
// Usage:
//
//	export LETTERMINT_TEAM_TOKEN="lm_team_..."
//	go run ./examples/team-api
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	lettermint "github.com/lettermint/lettermint-go/v3"
)

func main() {
	client, err := lettermint.New(lettermint.WithTeamToken(os.Getenv("LETTERMINT_TEAM_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	for domain, err := range client.Domains.Iterate(ctx, &lettermint.ListDomainsQuery{FilterStatus: lettermint.DomainStatusVerified}) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(domain.Domain, domain.Status)
	}

	stats, err := client.Stats.Retrieve(ctx, lettermint.GetStatsQuery{From: "2026-10-01", To: "2026-10-31"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", stats.Totals)
}
