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
	// Invites manages the pending invitations to the team.
	Invites *TeamInvitesService
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

// RetrieveRole returns a role.
func (s *TeamService) RetrieveRole(ctx context.Context, roleID string, options ...RequestOption) (*TeamRoleData, error) {
	return callJSON[TeamRoleData](ctx, s.t, opGetTeamRole, callArgs{label: "Team.RetrieveRole", path: []string{roleID}, options: requestOptions(options)})
}

// CreateRole creates a custom role. Custom roles need a plan that includes
// them, role names are unique within the team, and Permissions must include
// the permissions that the chosen permissions depend on.
func (s *TeamService) CreateRole(ctx context.Context, body CreateTeamRoleData, options ...RequestOption) (*TeamRoleData, error) {
	return callJSON[TeamRoleData](ctx, s.t, opCreateTeamRole, callArgs{label: "Team.CreateRole", body: body, options: requestOptions(options)})
}

// UpdateRole replaces the name, description and permissions of a custom
// role. Built-in roles cannot be changed.
func (s *TeamService) UpdateRole(ctx context.Context, roleID string, body UpdateTeamRoleData, options ...RequestOption) (*TeamRoleData, error) {
	return callJSON[TeamRoleData](ctx, s.t, opUpdateTeamRole, callArgs{label: "Team.UpdateRole", path: []string{roleID}, body: body, options: requestOptions(options)})
}

// DeleteRole deletes a custom role. Built-in roles cannot be deleted. Pass
// nil for no query parameters, or &DeleteTeamRoleQuery{ReplacementRoleID: id}
// to move the role's members, invitations and SCIM groups to another role;
// the replacement is required while the role is still assigned.
func (s *TeamService) DeleteRole(ctx context.Context, roleID string, query *DeleteTeamRoleQuery, options ...RequestOption) (*MessageResponse, error) {
	return callJSON[MessageResponse](ctx, s.t, opDeleteTeamRole, callArgs{label: "Team.DeleteRole", path: []string{roleID}, query: query, options: requestOptions(options)})
}

// DuplicateRole copies a role and its permissions under a new name. Without
// a Description, the copy keeps the description of the source role. The
// Owner role cannot be duplicated.
func (s *TeamService) DuplicateRole(ctx context.Context, roleID string, body DuplicateTeamRoleData, options ...RequestOption) (*TeamRoleData, error) {
	return callJSON[TeamRoleData](ctx, s.t, opDuplicateTeamRole, callArgs{label: "Team.DuplicateRole", path: []string{roleID}, body: body, options: requestOptions(options)})
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

// TeamInvitesService manages the pending invitations to the team. Needs a
// team token.
type TeamInvitesService struct{ service }

// List returns one page of pending invitations. Pass nil for no query
// parameters.
func (s *TeamInvitesService) List(ctx context.Context, query *ListTeamInvitesQuery, options ...RequestOption) (*ListTeamInvitesResponse, error) {
	return callJSON[ListTeamInvitesResponse](ctx, s.t, opListTeamInvites, callArgs{label: "Team.Invites.List", query: query, options: requestOptions(options)})
}

// Iterate yields every pending invitation, following next_cursor.
func (s *TeamInvitesService) Iterate(ctx context.Context, query *ListTeamInvitesQuery, options ...RequestOption) iter.Seq2[TeamInviteData, error] {
	return iterate[TeamInviteData](ctx, s.t, opListTeamInvites, callArgs{label: "Team.Invites.Iterate", query: query, options: requestOptions(options)})
}

// Retrieve returns a pending invitation.
func (s *TeamInvitesService) Retrieve(ctx context.Context, inviteID string, options ...RequestOption) (*TeamInviteData, error) {
	return callJSON[TeamInviteData](ctx, s.t, opGetTeamInvite, callArgs{label: "Team.Invites.Retrieve", path: []string{inviteID}, options: requestOptions(options)})
}

// Create invites a person to the team with a role and project access, and
// sends the invitation email. The team needs a free seat, and the Owner role
// cannot be assigned.
func (s *TeamInvitesService) Create(ctx context.Context, body CreateTeamInviteData, options ...RequestOption) (*TeamInviteData, error) {
	return callJSON[TeamInviteData](ctx, s.t, opCreateTeamInvite, callArgs{label: "Team.Invites.Create", body: body, options: requestOptions(options)})
}

// Update replaces the email address, role and project access of a pending
// invitation. A new invitation email is sent only when the address changes.
func (s *TeamInvitesService) Update(ctx context.Context, inviteID string, body UpdateTeamInviteData, options ...RequestOption) (*TeamInviteData, error) {
	return callJSON[TeamInviteData](ctx, s.t, opUpdateTeamInvite, callArgs{label: "Team.Invites.Update", path: []string{inviteID}, body: body, options: requestOptions(options)})
}

// Revoke revokes a pending invitation.
func (s *TeamInvitesService) Revoke(ctx context.Context, inviteID string, options ...RequestOption) (*MessageResponse, error) {
	return callJSON[MessageResponse](ctx, s.t, opDeleteTeamInvite, callArgs{label: "Team.Invites.Revoke", path: []string{inviteID}, options: requestOptions(options)})
}
