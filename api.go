package lettermint

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

type APIClient struct {
	client       *Client
	Domains      *DomainsService
	Messages     *MessagesService
	Projects     *ProjectsService
	Routes       *RoutesService
	Stats        *StatsService
	Suppressions *SuppressionsService
	Team         *TeamService
	Webhooks     *WebhooksService
}

func NewAPI(apiToken string, opts ...Option) (*APIClient, error) {
	client, err := newClient(apiToken, authSchemeBearer, opts...)
	if err != nil {
		return nil, err
	}

	return &APIClient{
		client:       client,
		Domains:      &DomainsService{client: client},
		Messages:     &MessagesService{client: client},
		Projects:     &ProjectsService{client: client},
		Routes:       &RoutesService{client: client},
		Stats:        &StatsService{client: client},
		Suppressions: &SuppressionsService{client: client},
		Team:         &TeamService{client: client},
		Webhooks:     &WebhooksService{client: client},
	}, nil
}

func (api *APIClient) Ping(ctx context.Context) (string, error) {
	return rawPing(api.client.doRaw(ctx, http.MethodGet, "/ping", nil))
}

func (c *Client) Ping(ctx context.Context) (string, error) {
	return rawPing(c.doRaw(ctx, http.MethodGet, "/ping", nil))
}

func (c *Client) SendBatch(ctx context.Context, payload SendBatchMailRequest) (SendBatchEmailResponse, error) {
	var out SendBatchEmailResponse
	err := c.doJSON(ctx, http.MethodPost, "/send/batch", nil, payload, &out)
	return out, err
}

type DomainsService struct{ client *Client }
type MessagesService struct{ client *Client }
type ProjectsService struct{ client *Client }
type RoutesService struct{ client *Client }
type StatsService struct{ client *Client }
type SuppressionsService struct{ client *Client }
type TeamService struct{ client *Client }
type WebhooksService struct{ client *Client }

func (s *DomainsService) List(ctx context.Context, query map[string]string) (DomainIndexResponse, error) {
	var out DomainIndexResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/domains", query, nil, &out)
	return out, err
}

func (s *DomainsService) Create(ctx context.Context, payload DomainStoreRequest) (DomainStoreResponse, error) {
	var out DomainStoreResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/domains", nil, payload, &out)
	return out, err
}

func (s *DomainsService) Retrieve(ctx context.Context, domainID string) (DomainShowResponse, error) {
	var out DomainShowResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/domains/"+segment(domainID), nil, nil, &out)
	return out, err
}

func (s *DomainsService) Delete(ctx context.Context, domainID string) (DomainDestroyResponse, error) {
	var out DomainDestroyResponse
	err := s.client.doJSON(ctx, http.MethodDelete, "/domains/"+segment(domainID), nil, nil, &out)
	return out, err
}

func (s *DomainsService) VerifyDNSRecords(ctx context.Context, domainID string) (DomainVerifyDNSRecordsResponse, error) {
	var out DomainVerifyDNSRecordsResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/domains/"+segment(domainID)+"/dns-records/verify", nil, nil, &out)
	return out, err
}

func (s *DomainsService) VerifyDNSRecord(ctx context.Context, domainID, recordID string) (DomainVerifySpecificDNSRecordResponse, error) {
	var out DomainVerifySpecificDNSRecordResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/domains/"+segment(domainID)+"/dns-records/"+segment(recordID)+"/verify", nil, nil, &out)
	return out, err
}

func (s *DomainsService) UpdateProjects(ctx context.Context, domainID string, payload DomainUpdateProjectsRequest) (DomainUpdateProjectsResponse, error) {
	var out DomainUpdateProjectsResponse
	err := s.client.doJSON(ctx, http.MethodPut, "/domains/"+segment(domainID)+"/projects", nil, payload, &out)
	return out, err
}

func (s *MessagesService) List(ctx context.Context, query map[string]string) (MessageIndexResponse, error) {
	var out MessageIndexResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/messages", query, nil, &out)
	return out, err
}

func (s *MessagesService) Retrieve(ctx context.Context, messageID string) (MessageShowResponse, error) {
	var out MessageShowResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/messages/"+segment(messageID), nil, nil, &out)
	return out, err
}

func (s *MessagesService) Events(ctx context.Context, messageID string) (MessageEventsResponse, error) {
	var out MessageEventsResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/messages/"+segment(messageID)+"/events", nil, nil, &out)
	return out, err
}

func (s *MessagesService) Source(ctx context.Context, messageID string) (string, error) {
	return s.client.doRaw(ctx, http.MethodGet, "/messages/"+segment(messageID)+"/source", nil)
}

func (s *MessagesService) HTML(ctx context.Context, messageID string) (string, error) {
	return s.client.doRaw(ctx, http.MethodGet, "/messages/"+segment(messageID)+"/html", nil)
}

func (s *MessagesService) Text(ctx context.Context, messageID string) (string, error) {
	return s.client.doRaw(ctx, http.MethodGet, "/messages/"+segment(messageID)+"/text", nil)
}

func (s *ProjectsService) List(ctx context.Context, query map[string]string) (ProjectIndexResponse, error) {
	var out ProjectIndexResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/projects", query, nil, &out)
	return out, err
}

func (s *ProjectsService) Create(ctx context.Context, payload ProjectStoreRequest) (ProjectStoreResponse, error) {
	var out ProjectStoreResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/projects", nil, payload, &out)
	return out, err
}

func (s *ProjectsService) Retrieve(ctx context.Context, projectID string) (ProjectShowResponse, error) {
	var out ProjectShowResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/projects/"+segment(projectID), nil, nil, &out)
	return out, err
}

func (s *ProjectsService) Update(ctx context.Context, projectID string, payload ProjectUpdateRequest) (ProjectUpdateResponse, error) {
	var out ProjectUpdateResponse
	err := s.client.doJSON(ctx, http.MethodPut, "/projects/"+segment(projectID), nil, payload, &out)
	return out, err
}

func (s *ProjectsService) Delete(ctx context.Context, projectID string) (ProjectDestroyResponse, error) {
	var out ProjectDestroyResponse
	err := s.client.doJSON(ctx, http.MethodDelete, "/projects/"+segment(projectID), nil, nil, &out)
	return out, err
}

func (s *ProjectsService) RotateToken(ctx context.Context, projectID string) (ProjectRotateTokenResponse, error) {
	var out ProjectRotateTokenResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/projects/"+segment(projectID)+"/rotate-token", nil, nil, &out)
	return out, err
}

func (s *ProjectsService) UpdateMembers(ctx context.Context, projectID string, payload ProjectUpdateMembersRequest) (ProjectUpdateMembersResponse, error) {
	var out ProjectUpdateMembersResponse
	err := s.client.doJSON(ctx, http.MethodPut, "/projects/"+segment(projectID)+"/members", nil, payload, &out)
	return out, err
}

func (s *ProjectsService) AddMember(ctx context.Context, projectID, teamMemberID string) (ProjectAddMemberResponse, error) {
	var out ProjectAddMemberResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/projects/"+segment(projectID)+"/members/"+segment(teamMemberID), nil, nil, &out)
	return out, err
}

func (s *ProjectsService) RemoveMember(ctx context.Context, projectID, teamMemberID string) (ProjectRemoveMemberResponse, error) {
	var out ProjectRemoveMemberResponse
	err := s.client.doJSON(ctx, http.MethodDelete, "/projects/"+segment(projectID)+"/members/"+segment(teamMemberID), nil, nil, &out)
	return out, err
}

func (s *ProjectsService) Routes(ctx context.Context, projectID string, query map[string]string) (RouteIndexResponse, error) {
	var out RouteIndexResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/projects/"+segment(projectID)+"/routes", query, nil, &out)
	return out, err
}

func (s *ProjectsService) CreateRoute(ctx context.Context, projectID string, payload RouteStoreRequest) (RouteStoreResponse, error) {
	var out RouteStoreResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/projects/"+segment(projectID)+"/routes", nil, payload, &out)
	return out, err
}

func (s *RoutesService) Retrieve(ctx context.Context, routeID string) (RouteShowResponse, error) {
	var out RouteShowResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/routes/"+segment(routeID), nil, nil, &out)
	return out, err
}

func (s *RoutesService) Update(ctx context.Context, routeID string, payload RouteUpdateRequest) (RouteUpdateResponse, error) {
	var out RouteUpdateResponse
	err := s.client.doJSON(ctx, http.MethodPut, "/routes/"+segment(routeID), nil, payload, &out)
	return out, err
}

func (s *RoutesService) Delete(ctx context.Context, routeID string) (RouteDestroyResponse, error) {
	var out RouteDestroyResponse
	err := s.client.doJSON(ctx, http.MethodDelete, "/routes/"+segment(routeID), nil, nil, &out)
	return out, err
}

func (s *RoutesService) VerifyInboundDomain(ctx context.Context, routeID string) (RouteVerifyInboundDomainResponse, error) {
	var out RouteVerifyInboundDomainResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/routes/"+segment(routeID)+"/verify-inbound-domain", nil, nil, &out)
	return out, err
}

func (s *StatsService) Retrieve(ctx context.Context, query map[string]string) (StatsIndexResponse, error) {
	var out StatsIndexResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/stats", query, nil, &out)
	return out, err
}

func (s *SuppressionsService) List(ctx context.Context, query map[string]string) (SuppressionIndexResponse, error) {
	var out SuppressionIndexResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/suppressions", query, nil, &out)
	return out, err
}

func (s *SuppressionsService) Create(ctx context.Context, payload SuppressionStoreRequest) (SuppressionStoreResponse, error) {
	var out SuppressionStoreResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/suppressions", nil, payload, &out)
	return out, err
}

func (s *SuppressionsService) Delete(ctx context.Context, suppressionID string) (SuppressionDestroyResponse, error) {
	var out SuppressionDestroyResponse
	err := s.client.doJSON(ctx, http.MethodDelete, "/suppressions/"+segment(suppressionID), nil, nil, &out)
	return out, err
}

func (s *TeamService) Retrieve(ctx context.Context) (TeamShowResponse, error) {
	var out TeamShowResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/team", nil, nil, &out)
	return out, err
}

func (s *TeamService) Update(ctx context.Context, payload TeamUpdateRequest) (TeamUpdateResponse, error) {
	var out TeamUpdateResponse
	err := s.client.doJSON(ctx, http.MethodPut, "/team", nil, payload, &out)
	return out, err
}

func (s *TeamService) Usage(ctx context.Context) (TeamUsageResponse, error) {
	var out TeamUsageResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/team/usage", nil, nil, &out)
	return out, err
}

func (s *TeamService) Members(ctx context.Context, query map[string]string) (TeamMembersResponse, error) {
	var out TeamMembersResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/team/members", query, nil, &out)
	return out, err
}

func (s *WebhooksService) List(ctx context.Context, query map[string]string) (WebhookIndexResponse, error) {
	var out WebhookIndexResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/webhooks", query, nil, &out)
	return out, err
}

func (s *WebhooksService) Create(ctx context.Context, payload WebhookStoreRequest) (WebhookStoreResponse, error) {
	var out WebhookStoreResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/webhooks", nil, payload, &out)
	return out, err
}

func (s *WebhooksService) Retrieve(ctx context.Context, webhookID string) (WebhookShowResponse, error) {
	var out WebhookShowResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/webhooks/"+segment(webhookID), nil, nil, &out)
	return out, err
}

func (s *WebhooksService) Update(ctx context.Context, webhookID string, payload WebhookUpdateRequest) (WebhookUpdateResponse, error) {
	var out WebhookUpdateResponse
	err := s.client.doJSON(ctx, http.MethodPut, "/webhooks/"+segment(webhookID), nil, payload, &out)
	return out, err
}

func (s *WebhooksService) Delete(ctx context.Context, webhookID string) (WebhookDestroyResponse, error) {
	var out WebhookDestroyResponse
	err := s.client.doJSON(ctx, http.MethodDelete, "/webhooks/"+segment(webhookID), nil, nil, &out)
	return out, err
}

func (s *WebhooksService) Test(ctx context.Context, webhookID string) (WebhookTestResponse, error) {
	var out WebhookTestResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/webhooks/"+segment(webhookID)+"/test", nil, nil, &out)
	return out, err
}

func (s *WebhooksService) RegenerateSecret(ctx context.Context, webhookID string) (WebhookRegenerateSecretResponse, error) {
	var out WebhookRegenerateSecretResponse
	err := s.client.doJSON(ctx, http.MethodPost, "/webhooks/"+segment(webhookID)+"/regenerate-secret", nil, nil, &out)
	return out, err
}

func (s *WebhooksService) Deliveries(ctx context.Context, webhookID string, query map[string]string) (WebhookDeliveriesResponse, error) {
	var out WebhookDeliveriesResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/webhooks/"+segment(webhookID)+"/deliveries", query, nil, &out)
	return out, err
}

func (s *WebhooksService) Delivery(ctx context.Context, webhookID, deliveryID string) (WebhookShowDeliveryResponse, error) {
	var out WebhookShowDeliveryResponse
	err := s.client.doJSON(ctx, http.MethodGet, "/webhooks/"+segment(webhookID)+"/deliveries/"+segment(deliveryID), nil, nil, &out)
	return out, err
}

func segment(value string) string {
	return url.PathEscape(value)
}

func rawPing(value string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}
