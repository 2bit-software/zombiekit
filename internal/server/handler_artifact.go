package server

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	artifactv1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/artifact/v1"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/artifact/v1/artifactv1connect"
	"github.com/2bit-software/zombiekit/internal/server/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ArtifactService struct {
	artifactv1connect.UnimplementedArtifactServiceHandler
	storage storage.ArtifactStorage
}

func NewArtifactService(storage storage.ArtifactStorage) *ArtifactService {
	return &ArtifactService{storage: storage}
}

func (s *ArtifactService) GetArtifact(
	ctx context.Context,
	req *connect.Request[artifactv1.GetArtifactRequest],
) (*connect.Response[artifactv1.GetArtifactResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	if msg.InitiativeId == "" || msg.Path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("initiative_id and path are required"))
	}

	initID, err := uuid.Parse(msg.InitiativeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid initiative_id"))
	}

	artifact, err := s.storage.Get(ctx, initID, msg.Path)
	if err != nil {
		if errors.Is(err, storage.ErrArtifactNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&artifactv1.GetArtifactResponse{
		Artifact: artifactToProto(artifact),
	}), nil
}

func (s *ArtifactService) SaveArtifact(
	ctx context.Context,
	req *connect.Request[artifactv1.SaveArtifactRequest],
) (*connect.Response[artifactv1.SaveArtifactResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	if msg.InitiativeId == "" || msg.Path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("initiative_id and path are required"))
	}

	initID, err := uuid.Parse(msg.InitiativeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid initiative_id"))
	}

	contentType := msg.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}

	artifact := &storage.Artifact{
		InitiativeID: initID,
		Path:         msg.Path,
		Content:      msg.Content,
		ContentType:  contentType,
	}

	if err := s.storage.Save(ctx, artifact); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	saved, err := s.storage.Get(ctx, initID, msg.Path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&artifactv1.SaveArtifactResponse{
		Artifact: artifactToProto(saved),
	}), nil
}

func (s *ArtifactService) ListArtifacts(
	ctx context.Context,
	req *connect.Request[artifactv1.ListArtifactsRequest],
) (*connect.Response[artifactv1.ListArtifactsResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	if msg.InitiativeId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("initiative_id is required"))
	}

	initID, err := uuid.Parse(msg.InitiativeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid initiative_id"))
	}

	artifacts, err := s.storage.List(ctx, initID, msg.PathPrefix)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &artifactv1.ListArtifactsResponse{
		Artifacts: make([]*artifactv1.Artifact, 0, len(artifacts)),
	}
	for _, a := range artifacts {
		resp.Artifacts = append(resp.Artifacts, artifactToProtoWithoutContent(a))
	}

	return connect.NewResponse(resp), nil
}

func artifactToProto(a *storage.Artifact) *artifactv1.Artifact {
	return &artifactv1.Artifact{
		InitiativeId: a.InitiativeID.String(),
		Path:         a.Path,
		Content:      a.Content,
		ContentType:  a.ContentType,
		SizeBytes:    a.SizeBytes,
		CreatedAt:    timestamppb.New(a.CreatedAt),
		UpdatedAt:    timestamppb.New(a.UpdatedAt),
	}
}

func artifactToProtoWithoutContent(a *storage.Artifact) *artifactv1.Artifact {
	return &artifactv1.Artifact{
		InitiativeId: a.InitiativeID.String(),
		Path:         a.Path,
		ContentType:  a.ContentType,
		SizeBytes:    a.SizeBytes,
		CreatedAt:    timestamppb.New(a.CreatedAt),
		UpdatedAt:    timestamppb.New(a.UpdatedAt),
	}
}
