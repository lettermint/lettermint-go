package lettermint

import (
	"context"
	"iter"
)

// ProjectsService manages projects. Needs a team token.
type ProjectsService struct {
	service
	// ReportForwarding manages the DMARC and complaint report forwarding of a project.
	ReportForwarding *ReportForwardingService
	// APITokens manages the Project API tokens of a project.
	APITokens *ProjectAPITokensService
	// TrackingDomain manages the custom tracking domain of a project.
	TrackingDomain *TrackingDomainService
}

// List returns one page of projects. Pass nil for no query parameters.
func (s *ProjectsService) List(ctx context.Context, query *ListProjectsQuery, options ...RequestOption) (*ListProjectsResponse, error) {
	return callJSON[ListProjectsResponse](ctx, s.t, opListProjects, callArgs{label: "Projects.List", query: query, options: requestOptions(options)})
}

// Iterate yields every project, following next_cursor.
func (s *ProjectsService) Iterate(ctx context.Context, query *ListProjectsQuery, options ...RequestOption) iter.Seq2[ProjectListData, error] {
	return iterate[ProjectListData](ctx, s.t, opListProjects, callArgs{label: "Projects.Iterate", query: query, options: requestOptions(options)})
}

// Create creates a project. The response holds its sending token once (APIToken).
func (s *ProjectsService) Create(ctx context.Context, body StoreProjectData, options ...RequestOption) (*ProjectCreatedData, error) {
	return callJSON[ProjectCreatedData](ctx, s.t, opCreateProject, callArgs{label: "Projects.Create", body: body, options: requestOptions(options)})
}

// Retrieve returns a project. Pass nil for no query parameters.
func (s *ProjectsService) Retrieve(ctx context.Context, projectID string, query *GetProjectQuery, options ...RequestOption) (*ProjectData, error) {
	return callJSON[ProjectData](ctx, s.t, opGetProject, callArgs{label: "Projects.Retrieve", path: []string{projectID}, query: query, options: requestOptions(options)})
}

// Update changes a project.
func (s *ProjectsService) Update(ctx context.Context, projectID string, body UpdateProjectData, options ...RequestOption) (*ProjectMutationResponse, error) {
	return callJSON[ProjectMutationResponse](ctx, s.t, opUpdateProject, callArgs{label: "Projects.Update", path: []string{projectID}, body: body, options: requestOptions(options)})
}

// Delete removes a project.
func (s *ProjectsService) Delete(ctx context.Context, projectID string, options ...RequestOption) (*MessageResponse, error) {
	return callJSON[MessageResponse](ctx, s.t, opDeleteProject, callArgs{label: "Projects.Delete", path: []string{projectID}, options: requestOptions(options)})
}

// RotateToken rotates the project's legacy sending token.
//
// Deprecated: the API marks this endpoint as legacy. Use APITokens to list,
// create and delete specific tokens.
func (s *ProjectsService) RotateToken(ctx context.Context, projectID string, options ...RequestOption) (*RotateProjectTokenResponse, error) {
	return callJSON[RotateProjectTokenResponse](ctx, s.t, opRotateProjectToken, callArgs{label: "Projects.RotateToken", path: []string{projectID}, options: requestOptions(options)})
}

// ReportForwardingService manages the DMARC and complaint report forwarding
// of a project. Needs a team token.
type ReportForwardingService struct{ service }

// Retrieve returns the report forwarding settings of a project.
func (s *ReportForwardingService) Retrieve(ctx context.Context, projectID string, options ...RequestOption) (*GetReportForwardingResponse, error) {
	return callJSON[GetReportForwardingResponse](ctx, s.t, opGetReportForwarding, callArgs{label: "Projects.ReportForwarding.Retrieve", path: []string{projectID}, options: requestOptions(options)})
}

// Update sets the report forwarding address of a project.
func (s *ReportForwardingService) Update(ctx context.Context, projectID string, body ReportForwardingRequest, options ...RequestOption) (*UpdateReportForwardingResponse, error) {
	return callJSON[UpdateReportForwardingResponse](ctx, s.t, opUpdateReportForwarding, callArgs{label: "Projects.ReportForwarding.Update", path: []string{projectID}, body: body, options: requestOptions(options)})
}

// Delete disables report forwarding (HTTP 204).
func (s *ReportForwardingService) Delete(ctx context.Context, projectID string, options ...RequestOption) error {
	return callEmpty(ctx, s.t, opDeleteReportForwarding, callArgs{label: "Projects.ReportForwarding.Delete", path: []string{projectID}, options: requestOptions(options)})
}

// Verify confirms the forwarding address with the code that was sent to it.
func (s *ReportForwardingService) Verify(ctx context.Context, projectID string, body VerifyReportForwardingRequest, options ...RequestOption) (*VerifyReportForwardingResponse, error) {
	return callJSON[VerifyReportForwardingResponse](ctx, s.t, opVerifyReportForwarding, callArgs{label: "Projects.ReportForwarding.Verify", path: []string{projectID}, body: body, options: requestOptions(options)})
}

// ResendCode sends a new verification code to the forwarding address.
func (s *ReportForwardingService) ResendCode(ctx context.Context, projectID string, options ...RequestOption) (*ResendReportForwardingCodeResponse, error) {
	return callJSON[ResendReportForwardingCodeResponse](ctx, s.t, opResendReportForwardingCode, callArgs{label: "Projects.ReportForwarding.ResendCode", path: []string{projectID}, options: requestOptions(options)})
}

// ProjectAPITokensService manages the Project API tokens of a project. Needs
// a team token.
type ProjectAPITokensService struct{ service }

// List returns one page of the project's API tokens, without their values.
// Pass nil for no query parameters.
func (s *ProjectAPITokensService) List(ctx context.Context, projectID string, query *ListProjectApiTokensQuery, options ...RequestOption) (*ListProjectApiTokensResponse, error) {
	return callJSON[ListProjectApiTokensResponse](ctx, s.t, opListProjectApiTokens, callArgs{label: "Projects.APITokens.List", path: []string{projectID}, query: query, options: requestOptions(options)})
}

// Iterate yields every API token of the project, following next_cursor.
func (s *ProjectAPITokensService) Iterate(ctx context.Context, projectID string, query *ListProjectApiTokensQuery, options ...RequestOption) iter.Seq2[ProjectApiToken, error] {
	return iterate[ProjectApiToken](ctx, s.t, opListProjectApiTokens, callArgs{label: "Projects.APITokens.Iterate", path: []string{projectID}, query: query, options: requestOptions(options)})
}

// Create creates a Project API token. The response holds its value once
// (Data.Token): use it to send through the API, or as the SMTP password when
// SMTP is enabled for the project (SMTP is nil otherwise). Set ShortToken for
// devices that limit the password length.
func (s *ProjectAPITokensService) Create(ctx context.Context, projectID string, body StoreProjectApiTokenData, options ...RequestOption) (*ProjectApiTokenCreatedResponse, error) {
	return callJSON[ProjectApiTokenCreatedResponse](ctx, s.t, opCreateProjectApiToken, callArgs{label: "Projects.APITokens.Create", path: []string{projectID}, body: body, options: requestOptions(options)})
}

// Delete deletes a Project API token.
func (s *ProjectAPITokensService) Delete(ctx context.Context, projectID, tokenID string, options ...RequestOption) (*MessageResponse, error) {
	return callJSON[MessageResponse](ctx, s.t, opDeleteProjectApiToken, callArgs{label: "Projects.APITokens.Delete", path: []string{projectID, tokenID}, options: requestOptions(options)})
}

// TrackingDomainService manages the custom tracking domain of a project: the
// domain that open and click tracking links use instead of the Lettermint
// tracking domain. Needs a team token.
type TrackingDomainService struct{ service }

// Retrieve returns the custom tracking domain of a project and whether its
// CNAME record is verified. Domain is nil when the project uses the
// Lettermint tracking domain.
func (s *TrackingDomainService) Retrieve(ctx context.Context, projectID string, options ...RequestOption) (*ProjectTrackingDomain, error) {
	return callJSON[ProjectTrackingDomain](ctx, s.t, opGetProjectTrackingDomain, callArgs{label: "Projects.TrackingDomain.Retrieve", path: []string{projectID}, options: requestOptions(options)})
}

// Update sets or replaces the custom tracking domain of a project. Add a
// CNAME record for the domain that points to Data.CnameTarget. Lettermint
// checks the record in the background, or call Verify to check it now.
// Tracking links keep using the Lettermint tracking domain until the domain is
// verified. Needs a plan with custom tracking domains.
func (s *TrackingDomainService) Update(ctx context.Context, projectID string, body UpdateProjectTrackingDomainData, options ...RequestOption) (*ProjectTrackingDomainMutationResponse, error) {
	return callJSON[ProjectTrackingDomainMutationResponse](ctx, s.t, opUpdateProjectTrackingDomain, callArgs{label: "Projects.TrackingDomain.Update", path: []string{projectID}, body: body, options: requestOptions(options)})
}

// Delete removes the custom tracking domain (HTTP 204). Tracking links use the
// Lettermint tracking domain again.
func (s *TrackingDomainService) Delete(ctx context.Context, projectID string, options ...RequestOption) error {
	return callEmpty(ctx, s.t, opDeleteProjectTrackingDomain, callArgs{label: "Projects.TrackingDomain.Delete", path: []string{projectID}, options: requestOptions(options)})
}

// Verify checks the CNAME record of the custom tracking domain now. When the
// record is missing, points to another target or the DNS lookup failed, the
// API answers 422 and Verify returns a *ValidationError. Its Body then holds
// a ProjectTrackingDomainVerificationResponse (Data.TemporaryError is true
// when the lookup failed; try again later). The API also answers 422 when the
// project has no custom tracking domain.
func (s *TrackingDomainService) Verify(ctx context.Context, projectID string, options ...RequestOption) (*ProjectTrackingDomainVerificationResponse, error) {
	return callJSON[ProjectTrackingDomainVerificationResponse](ctx, s.t, opVerifyProjectTrackingDomain, callArgs{label: "Projects.TrackingDomain.Verify", path: []string{projectID}, options: requestOptions(options)})
}
