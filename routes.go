package lettermint

import (
	"context"
	"iter"
)

// RoutesService manages the routes of a project. Needs a team token.
type RoutesService struct{ service }

// List returns one page of the routes of a project. Pass nil for no query
// parameters.
func (s *RoutesService) List(ctx context.Context, projectID string, query *ListRoutesQuery, options ...RequestOption) (*ListRoutesResponse, error) {
	return callJSON[ListRoutesResponse](ctx, s.t, opListRoutes, callArgs{label: "Routes.List", path: []string{projectID}, query: query, options: requestOptions(options)})
}

// Iterate yields every route of a project, following next_cursor.
func (s *RoutesService) Iterate(ctx context.Context, projectID string, query *ListRoutesQuery, options ...RequestOption) iter.Seq2[RouteListData, error] {
	return iterate[RouteListData](ctx, s.t, opListRoutes, callArgs{label: "Routes.Iterate", path: []string{projectID}, query: query, options: requestOptions(options)})
}

// Create adds a route to a project.
func (s *RoutesService) Create(ctx context.Context, projectID string, body StoreRouteData, options ...RequestOption) (*RouteMutationResponse, error) {
	return callJSON[RouteMutationResponse](ctx, s.t, opCreateRoute, callArgs{label: "Routes.Create", path: []string{projectID}, body: body, options: requestOptions(options)})
}

// Retrieve returns a route. Pass nil for no query parameters.
func (s *RoutesService) Retrieve(ctx context.Context, routeID string, query *GetRouteQuery, options ...RequestOption) (*RouteData, error) {
	return callJSON[RouteData](ctx, s.t, opGetRoute, callArgs{label: "Routes.Retrieve", path: []string{routeID}, query: query, options: requestOptions(options)})
}

// Update changes a route.
func (s *RoutesService) Update(ctx context.Context, routeID string, body UpdateRouteData, options ...RequestOption) (*RouteMutationResponse, error) {
	return callJSON[RouteMutationResponse](ctx, s.t, opUpdateRoute, callArgs{label: "Routes.Update", path: []string{routeID}, body: body, options: requestOptions(options)})
}

// Delete removes a route.
func (s *RoutesService) Delete(ctx context.Context, routeID string, options ...RequestOption) (*MessageResponse, error) {
	return callJSON[MessageResponse](ctx, s.t, opDeleteRoute, callArgs{label: "Routes.Delete", path: []string{routeID}, options: requestOptions(options)})
}

// VerifyInboundDomain checks the DNS records of the route's inbound domain.
func (s *RoutesService) VerifyInboundDomain(ctx context.Context, routeID string, options ...RequestOption) (*InboundDomainVerificationResponse, error) {
	return callJSON[InboundDomainVerificationResponse](ctx, s.t, opVerifyRouteInboundDomain, callArgs{label: "Routes.VerifyInboundDomain", path: []string{routeID}, options: requestOptions(options)})
}
