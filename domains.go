package lettermint

import (
	"context"
	"iter"
)

// DomainsService manages sending domains. Needs a team token.
type DomainsService struct{ service }

// List returns one page of domains. Pass nil for no query parameters.
func (s *DomainsService) List(ctx context.Context, query *ListDomainsQuery, options ...RequestOption) (*ListDomainsResponse, error) {
	return callJSON[ListDomainsResponse](ctx, s.t, opListDomains, callArgs{label: "Domains.List", query: query, options: requestOptions(options)})
}

// Iterate yields every domain, following next_cursor. It requests the next
// page only when the loop gets to it, and yields an error at most once.
//
//	for domain, err := range client.Domains.Iterate(ctx, nil) {
//		if err != nil {
//			return err
//		}
//		fmt.Println(domain.Domain)
//	}
func (s *DomainsService) Iterate(ctx context.Context, query *ListDomainsQuery, options ...RequestOption) iter.Seq2[DomainListData, error] {
	return iterate[DomainListData](ctx, s.t, opListDomains, callArgs{label: "Domains.Iterate", query: query, options: requestOptions(options)})
}

// Create adds a sending domain.
func (s *DomainsService) Create(ctx context.Context, body StoreDomainData, options ...RequestOption) (*DomainData, error) {
	return callJSON[DomainData](ctx, s.t, opCreateDomain, callArgs{label: "Domains.Create", body: body, options: requestOptions(options)})
}

// Retrieve returns a domain. Pass nil for no query parameters, or
// &GetDomainQuery{Include: []GetDomainQueryIncludeItem{GetDomainQueryIncludeItemDNSRecords}}.
func (s *DomainsService) Retrieve(ctx context.Context, domainID string, query *GetDomainQuery, options ...RequestOption) (*DomainData, error) {
	return callJSON[DomainData](ctx, s.t, opGetDomain, callArgs{label: "Domains.Retrieve", path: []string{domainID}, query: query, options: requestOptions(options)})
}

// Delete removes a domain.
func (s *DomainsService) Delete(ctx context.Context, domainID string, options ...RequestOption) (*MessageResponse, error) {
	return callJSON[MessageResponse](ctx, s.t, opDeleteDomain, callArgs{label: "Domains.Delete", path: []string{domainID}, options: requestOptions(options)})
}

// VerifyDNSRecords checks every DNS record of the domain.
func (s *DomainsService) VerifyDNSRecords(ctx context.Context, domainID string, options ...RequestOption) (*DnsVerificationSuccessResponse, error) {
	return callJSON[DnsVerificationSuccessResponse](ctx, s.t, opVerifyDomainDnsRecords, callArgs{label: "Domains.VerifyDNSRecords", path: []string{domainID}, options: requestOptions(options)})
}

// VerifyDNSRecord checks one DNS record of the domain.
func (s *DomainsService) VerifyDNSRecord(ctx context.Context, domainID, recordID string, options ...RequestOption) (*MessageResponse, error) {
	return callJSON[MessageResponse](ctx, s.t, opVerifyDomainDnsRecord, callArgs{label: "Domains.VerifyDNSRecord", path: []string{domainID, recordID}, options: requestOptions(options)})
}

// UpdateProjects replaces the projects that may send from the domain.
func (s *DomainsService) UpdateProjects(ctx context.Context, domainID string, body UpdateDomainProjectsData, options ...RequestOption) (*DomainMutationResponse, error) {
	return callJSON[DomainMutationResponse](ctx, s.t, opUpdateDomainProjects, callArgs{label: "Domains.UpdateProjects", path: []string{domainID}, body: body, options: requestOptions(options)})
}

// StartDkimUpgrade moves a domain with a legacy_txt DKIM record to managed
// DKIM. The response lists two new DKIM CNAME records: add them to your DNS.
// The current DKIM TXT record keeps signing until the upgrade completes. When
// Data.DkimTransition reports DkimTransitionPhaseReadyToActivate, call
// ActivateDkimUpgrade. Calling it again returns the upgrade in progress. A
// domain that already uses managed DKIM is returned unchanged. The API
// returns a *ValidationError (DKIM_UPGRADE_UNAVAILABLE) when the domain
// cannot be upgraded.
func (s *DomainsService) StartDkimUpgrade(ctx context.Context, domainID string, options ...RequestOption) (*DomainMutationResponse, error) {
	return callJSON[DomainMutationResponse](ctx, s.t, opStartDomainDkimUpgrade, callArgs{label: "Domains.StartDkimUpgrade", path: []string{domainID}, options: requestOptions(options)})
}

// CancelDkimUpgrade cancels a managed DKIM upgrade and removes its DKIM
// CNAME records. The domain keeps its DKIM TXT record. Without an upgrade in
// progress nothing changes, and a key rotation is never cancelled. The API
// returns a *ValidationError (DKIM_UPGRADE_IN_GRACE_PERIOD) once the grace
// period has started.
func (s *DomainsService) CancelDkimUpgrade(ctx context.Context, domainID string, options ...RequestOption) (*DomainMutationResponse, error) {
	return callJSON[DomainMutationResponse](ctx, s.t, opCancelDomainDkimUpgrade, callArgs{label: "Domains.CancelDkimUpgrade", path: []string{domainID}, options: requestOptions(options)})
}

// ActivateDkimUpgrade checks both DKIM CNAME records again and starts a grace
// period in which both keys are published. The upgrade completes at
// Data.DkimTransition's CompletesAfter, and DkimMode then changes to
// managed_cname; after that you can remove the old DKIM TXT record. Calling it
// again during the grace period, or after the upgrade completed, changes
// nothing. The API returns a *ValidationError (NO_DKIM_UPGRADE or
// DKIM_UPGRADE_NOT_READY) when there is no upgrade or it is not ready.
func (s *DomainsService) ActivateDkimUpgrade(ctx context.Context, domainID string, options ...RequestOption) (*DomainMutationResponse, error) {
	return callJSON[DomainMutationResponse](ctx, s.t, opActivateDomainDkimUpgrade, callArgs{label: "Domains.ActivateDkimUpgrade", path: []string{domainID}, options: requestOptions(options)})
}

// RotateDkimKey starts a key rotation for a domain with managed DKIM. The
// DKIM CNAME records stay the same, so no DNS change is needed. Lettermint
// also rotates keys on a schedule, so you only need this to replace a key
// early. Calling it again during a rotation returns the rotation in progress.
// The API returns a *ValidationError (DKIM_ROTATION_UNAVAILABLE) when the
// domain cannot be rotated.
func (s *DomainsService) RotateDkimKey(ctx context.Context, domainID string, options ...RequestOption) (*DomainMutationResponse, error) {
	return callJSON[DomainMutationResponse](ctx, s.t, opRotateDomainDkimKey, callArgs{label: "Domains.RotateDkimKey", path: []string{domainID}, options: requestOptions(options)})
}
