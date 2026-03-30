package handlers

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	profilev1 "github.com/2bit-software/zombiekit/gen/zombiekit/brains/profile/v1"
	"github.com/2bit-software/zombiekit/gen/zombiekit/brains/profile/v1/profilev1connect"
	"github.com/2bit-software/zombiekit/internal/server/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProfileService struct {
	profilev1connect.UnimplementedProfileServiceHandler
	storage storage.ProfileStorage
}

func NewProfileService(storage storage.ProfileStorage) *ProfileService {
	return &ProfileService{storage: storage}
}

func (s *ProfileService) ListProfiles(
	ctx context.Context,
	req *connect.Request[profilev1.ListProfilesRequest],
) (*connect.Response[profilev1.ListProfilesResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("profile storage not configured"))
	}

	profiles, err := s.storage.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &profilev1.ListProfilesResponse{
		Profiles: make([]*profilev1.Profile, 0, len(profiles)),
	}

	for _, p := range profiles {
		resp.Profiles = append(resp.Profiles, storageToProto(p))
	}

	return connect.NewResponse(resp), nil
}

func (s *ProfileService) GetProfile(
	ctx context.Context,
	req *connect.Request[profilev1.GetProfileRequest],
) (*connect.Response[profilev1.GetProfileResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("profile storage not configured"))
	}

	p, err := s.storage.Get(ctx, req.Msg.Name)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&profilev1.GetProfileResponse{
		Profile: storageToProto(p),
	}), nil
}

func (s *ProfileService) SaveProfile(
	ctx context.Context,
	req *connect.Request[profilev1.SaveProfileRequest],
) (*connect.Response[profilev1.SaveProfileResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("profile storage not configured"))
	}

	msg := req.Msg
	if msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	existing, err := s.storage.Get(ctx, msg.Name)
	if err != nil && !errors.Is(err, storage.ErrProfileNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if existing != nil && !msg.Overwrite {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("profile already exists"))
	}

	profile := &storage.Profile{
		Name:     msg.Name,
		Content:  msg.Content,
		Location: protoToLocation(msg.Location),
	}
	if existing != nil {
		profile.ID = existing.ID
		profile.CreatedAt = existing.CreatedAt
	}

	if err := s.storage.Save(ctx, profile); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	saved, err := s.storage.Get(ctx, msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&profilev1.SaveProfileResponse{
		Profile: storageToProto(saved),
	}), nil
}

func (s *ProfileService) ComposeProfile(
	ctx context.Context,
	req *connect.Request[profilev1.ComposeProfileRequest],
) (*connect.Response[profilev1.ComposeProfileResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("profile storage not configured"))
	}

	if len(req.Msg.ProfileNames) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile_names is required"))
	}

	profiles, err := s.storage.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	profileMap := make(map[string]*storage.Profile)
	for _, p := range profiles {
		profileMap[p.Name] = p
	}

	var composed string
	var resolved []string

	seen := make(map[string]bool)
	for _, name := range req.Msg.ProfileNames {
		if err := s.composeRecursive(name, profileMap, seen, &composed, &resolved); err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
	}

	return connect.NewResponse(&profilev1.ComposeProfileResponse{
		ComposedContent:  composed,
		ResolvedProfiles: resolved,
	}), nil
}

func (s *ProfileService) composeRecursive(
	name string,
	profiles map[string]*storage.Profile,
	seen map[string]bool,
	composed *string,
	resolved *[]string,
) error {
	if seen[name] {
		return nil
	}

	p, ok := profiles[name]
	if !ok {
		return errors.New("profile not found: " + name)
	}

	for _, dep := range p.Dependencies {
		if err := s.composeRecursive(dep, profiles, seen, composed, resolved); err != nil {
			return err
		}
	}

	seen[name] = true
	if *composed != "" {
		*composed += "\n\n"
	}
	*composed += p.Content
	*resolved = append(*resolved, name)

	return nil
}

func storageToProto(p *storage.Profile) *profilev1.Profile {
	return &profilev1.Profile{
		Name:         p.Name,
		Content:      p.Content,
		Domains:      p.Domains,
		Dependencies: p.Dependencies,
		Location:     locationToProto(p.Location),
		CreatedAt:    timestamppb.New(p.CreatedAt),
		UpdatedAt:    timestamppb.New(p.UpdatedAt),
	}
}

func locationToProto(loc storage.ProfileLocation) profilev1.ProfileLocation {
	switch loc {
	case storage.ProfileLocationLocal:
		return profilev1.ProfileLocation_PROFILE_LOCATION_LOCAL
	case storage.ProfileLocationGlobal:
		return profilev1.ProfileLocation_PROFILE_LOCATION_GLOBAL
	default:
		return profilev1.ProfileLocation_PROFILE_LOCATION_UNSPECIFIED
	}
}

func protoToLocation(loc profilev1.ProfileLocation) storage.ProfileLocation {
	switch loc {
	case profilev1.ProfileLocation_PROFILE_LOCATION_LOCAL:
		return storage.ProfileLocationLocal
	case profilev1.ProfileLocation_PROFILE_LOCATION_GLOBAL:
		return storage.ProfileLocationGlobal
	default:
		return storage.ProfileLocationGlobal
	}
}
