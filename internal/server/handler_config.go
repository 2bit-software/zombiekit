package server

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	configv1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/config/v1"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/config/v1/configv1connect"
	"github.com/2bit-software/zombiekit/internal/server/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConfigService struct {
	configv1connect.UnimplementedConfigServiceHandler
	storage storage.ConfigStorage
}

func NewConfigService(storage storage.ConfigStorage) *ConfigService {
	return &ConfigService{storage: storage}
}

func (s *ConfigService) GetConfig(
	ctx context.Context,
	req *connect.Request[configv1.GetConfigRequest],
) (*connect.Response[configv1.GetConfigResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	entries, err := s.storage.Get(ctx, req.Msg.Keys)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &configv1.GetConfigResponse{
		Entries: make([]*configv1.Config, 0, len(entries)),
	}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, &configv1.Config{
			Key:       e.Key,
			Value:     e.Value,
			UpdatedAt: timestamppb.New(e.UpdatedAt),
		})
	}

	return connect.NewResponse(resp), nil
}

func (s *ConfigService) UpdateConfig(
	ctx context.Context,
	req *connect.Request[configv1.UpdateConfigRequest],
) (*connect.Response[configv1.UpdateConfigResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	entries := make([]*storage.ConfigEntry, 0, len(msg.Entries))
	for _, e := range msg.Entries {
		entries = append(entries, &storage.ConfigEntry{
			Key:   e.Key,
			Value: e.Value,
		})
	}

	if err := s.storage.Set(ctx, entries); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		keys = append(keys, e.Key)
	}

	updated, err := s.storage.Get(ctx, keys)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &configv1.UpdateConfigResponse{
		Entries: make([]*configv1.Config, 0, len(updated)),
	}
	for _, e := range updated {
		resp.Entries = append(resp.Entries, &configv1.Config{
			Key:       e.Key,
			Value:     e.Value,
			UpdatedAt: timestamppb.New(e.UpdatedAt),
		})
	}

	return connect.NewResponse(resp), nil
}
