package management

import (
	"context"

	"github.com/caos/zitadel/internal/api/authz"
	obj_grpc "github.com/caos/zitadel/internal/api/grpc/object"
	"github.com/caos/zitadel/internal/api/grpc/user"
	mgmt_pb "github.com/caos/zitadel/pkg/grpc/management"
)

func (s *Server) GetUserGrantByID(ctx context.Context, req *mgmt_pb.GetUserGrantByIDRequest) (*mgmt_pb.GetUserGrantByIDResponse, error) {
	//TODO: resource owner
	grant, err := s.query.UserGrantByID(ctx, req.GrantId)
	if err != nil {
		return nil, err
	}
	return &mgmt_pb.GetUserGrantByIDResponse{
		UserGrant: user.UserGrantToPb(s.assetAPIPrefix, grant),
	}, nil
}

func (s *Server) ListUserGrants(ctx context.Context, req *mgmt_pb.ListUserGrantRequest) (*mgmt_pb.ListUserGrantResponse, error) {
	queries, err := ListUserGrantsRequestToQuery(ctx, req)
	if err != nil {
		return nil, err
	}
	res, err := s.query.UserGrants(ctx, queries)
	if err != nil {
		return nil, err
	}
	return &mgmt_pb.ListUserGrantResponse{
		Result: user.UserGrantsToPb(s.assetAPIPrefix, res.UserGrants),
		Details: obj_grpc.ToListDetails(
			res.Count,
			res.Sequence,
			res.Timestamp,
		),
	}, nil
}

func (s *Server) AddUserGrant(ctx context.Context, req *mgmt_pb.AddUserGrantRequest) (*mgmt_pb.AddUserGrantResponse, error) {
	grant := AddUserGrantRequestToDomain(req)
	if err := checkExplicitProjectPermission(ctx, grant.ProjectGrantID, grant.ProjectID); err != nil {
		return nil, err
	}
	grant, err := s.command.AddUserGrant(ctx, grant, authz.GetCtxData(ctx).OrgID)
	if err != nil {
		return nil, err
	}
	return &mgmt_pb.AddUserGrantResponse{
		UserGrantId: grant.AggregateID,
		Details: obj_grpc.AddToDetailsPb(
			grant.Sequence,
			grant.ChangeDate,
			grant.ResourceOwner,
		),
	}, nil
}

func (s *Server) UpdateUserGrant(ctx context.Context, req *mgmt_pb.UpdateUserGrantRequest) (*mgmt_pb.UpdateUserGrantResponse, error) {
	grant, err := s.command.ChangeUserGrant(ctx, UpdateUserGrantRequestToDomain(req), authz.GetCtxData(ctx).OrgID)
	if err != nil {
		return nil, err
	}
	return &mgmt_pb.UpdateUserGrantResponse{
		Details: obj_grpc.ChangeToDetailsPb(
			grant.Sequence,
			grant.ChangeDate,
			grant.ResourceOwner,
		),
	}, nil
}

func (s *Server) DeactivateUserGrant(ctx context.Context, req *mgmt_pb.DeactivateUserGrantRequest) (*mgmt_pb.DeactivateUserGrantResponse, error) {
	objectDetails, err := s.command.DeactivateUserGrant(ctx, req.GrantId, authz.GetCtxData(ctx).OrgID)
	if err != nil {
		return nil, err
	}
	return &mgmt_pb.DeactivateUserGrantResponse{
		Details: obj_grpc.DomainToChangeDetailsPb(objectDetails),
	}, nil
}

func (s *Server) ReactivateUserGrant(ctx context.Context, req *mgmt_pb.ReactivateUserGrantRequest) (*mgmt_pb.ReactivateUserGrantResponse, error) {
	objectDetails, err := s.command.ReactivateUserGrant(ctx, req.GrantId, authz.GetCtxData(ctx).OrgID)
	if err != nil {
		return nil, err
	}
	return &mgmt_pb.ReactivateUserGrantResponse{
		Details: obj_grpc.DomainToChangeDetailsPb(objectDetails),
	}, nil
}

func (s *Server) RemoveUserGrant(ctx context.Context, req *mgmt_pb.RemoveUserGrantRequest) (*mgmt_pb.RemoveUserGrantResponse, error) {
	objectDetails, err := s.command.RemoveUserGrant(ctx, req.GrantId, authz.GetCtxData(ctx).OrgID)
	if err != nil {
		return nil, err
	}
	return &mgmt_pb.RemoveUserGrantResponse{
		Details: obj_grpc.DomainToChangeDetailsPb(objectDetails),
	}, nil
}

func (s *Server) BulkRemoveUserGrant(ctx context.Context, req *mgmt_pb.BulkRemoveUserGrantRequest) (*mgmt_pb.BulkRemoveUserGrantResponse, error) {
	err := s.command.BulkRemoveUserGrant(ctx, req.GrantId, authz.GetCtxData(ctx).OrgID)
	if err != nil {
		return nil, err
	}
	return &mgmt_pb.BulkRemoveUserGrantResponse{}, nil
}
