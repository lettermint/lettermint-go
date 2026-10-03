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
// Deprecated: the API marks this endpoint as legacy.
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
