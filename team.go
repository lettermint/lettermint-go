package lettermint

import (
	"context"
	"iter"
)

// StatsService reads sending statistics. Needs a team token.
type StatsService struct{ service }

// Retrieve returns daily statistics between From and To (Y-m-d, at most 90 days).
func (s *StatsService) Retrieve(ctx context.Context, query GetStatsQuery, options ...RequestOption) (*StatsData, error) {
	return callJSON[StatsData](ctx, s.t, opGetStats, callArgs{label: "Stats.Retrieve", query: &query, options: requestOptions(options)})
}

// SuppressionsService manages the suppression list. Needs a team token.
type SuppressionsService struct{ service }

// List returns one page of suppressions. Pass nil for no query parameters.
func (s *SuppressionsService) List(ctx context.Context, query *ListSuppressionsQuery, options ...RequestOption) (*ListSuppressionsResponse, error) {
	return callJSON[ListSuppressionsResponse](ctx, s.t, opListSuppressions, callArgs{label: "Suppressions.List", query: query, options: requestOptions(options)})
}

// Iterate yields every suppression, following next_cursor.
func (s *SuppressionsService) Iterate(ctx context.Context, query *ListSuppressionsQuery, options ...RequestOption) iter.Seq2[SuppressedRecipientData, error] {
	return iterate[SuppressedRecipientData](ctx, s.t, opListSuppressions, callArgs{label: "Suppressions.Iterate", query: query, options: requestOptions(options)})
}

// Create adds addresses or domains to the suppression list.
func (s *SuppressionsService) Create(ctx context.Context, body StoreSuppressionData, options ...RequestOption) (*SuppressionStoreResponse, error) {
	return callJSON[SuppressionStoreResponse](ctx, s.t, opCreateSuppressions, callArgs{label: "Suppressions.Create", body: body, options: requestOptions(options)})
}

// Delete removes an entry from the suppression list.
func (s *SuppressionsService) Delete(ctx context.Context, suppressionID string, options ...RequestOption) (*DeleteSuppressionResponse, error) {
	return callJSON[DeleteSuppressionResponse](ctx, s.t, opDeleteSuppression, callArgs{label: "Suppressions.Delete", path: []string{suppressionID}, options: requestOptions(options)})
}

// TeamService manages the team of the token. Needs a team token.
type TeamService struct {
	service
	// Members manages the members of the team.
	Members *TeamMembersService
}

// Retrieve returns the team. Pass nil for no query parameters.
func (s *TeamService) Retrieve(ctx context.Context, query *GetTeamQuery, options ...RequestOption) (*TeamData, error) {
	return callJSON[TeamData](ctx, s.t, opGetTeam, callArgs{label: "Team.Retrieve", query: query, options: requestOptions(options)})
}

// Update changes the team.
func (s *TeamService) Update(ctx context.Context, body UpdateTeamData, options ...RequestOption) (*TeamMutationResponse, error) {
	return callJSON[TeamMutationResponse](ctx, s.t, opUpdateTeam, callArgs{label: "Team.Update", body: body, options: requestOptions(options)})
}

// Usage returns the usage of the current and previous billing periods.
func (s *TeamService) Usage(ctx context.Context, options ...RequestOption) (*TeamUsageDetailData, error) {
	return callJSON[TeamUsageDetailData](ctx, s.t, opGetTeamUsage, callArgs{label: "Team.Usage", options: requestOptions(options)})
}

// Roles returns the roles that can be assigned to members.
func (s *TeamService) Roles(ctx context.Context, options ...RequestOption) (*TeamRoleListResponse, error) {
	return callJSON[TeamRoleListResponse](ctx, s.t, opListTeamRoles, callArgs{label: "Team.Roles", options: requestOptions(options)})
}

// TeamMembersService manages the members of the team. Needs a team token.
type TeamMembersService struct{ service }

// List returns one page of team members. Pass nil for no query parameters.
func (s *TeamMembersService) List(ctx context.Context, query *ListTeamMembersQuery, options ...RequestOption) (*ListTeamMembersResponse, error) {
	return callJSON[ListTeamMembersResponse](ctx, s.t, opListTeamMembers, callArgs{label: "Team.Members.List", query: query, options: requestOptions(options)})
}

// Iterate yields every team member, following next_cursor.
func (s *TeamMembersService) Iterate(ctx context.Context, query *ListTeamMembersQuery, options ...RequestOption) iter.Seq2[TeamMemberData, error] {
	return iterate[TeamMemberData](ctx, s.t, opListTeamMembers, callArgs{label: "Team.Members.Iterate", query: query, options: requestOptions(options)})
}

// Retrieve returns a team member.
func (s *TeamMembersService) Retrieve(ctx context.Context, userID string, options ...RequestOption) (*TeamMemberData, error) {
	return callJSON[TeamMemberData](ctx, s.t, opGetTeamMember, callArgs{label: "Team.Members.Retrieve", path: []string{userID}, options: requestOptions(options)})
}

// UpdateAssignment changes a member's role and project access.
func (s *TeamMembersService) UpdateAssignment(ctx context.Context, userID string, body UpdateTeamMemberAssignmentData, options ...RequestOption) (*TeamMemberData, error) {
	return callJSON[TeamMemberData](ctx, s.t, opUpdateTeamMemberAssignment, callArgs{label: "Team.Members.UpdateAssignment", path: []string{userID}, body: body, options: requestOptions(options)})
}
