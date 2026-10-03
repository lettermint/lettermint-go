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
