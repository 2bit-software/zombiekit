package server

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	workflowv1 "github.com/2bit-software/zombiekit/gen/zombiekit/brains/workflow/v1"
	"github.com/2bit-software/zombiekit/gen/zombiekit/brains/workflow/v1/workflowv1connect"
	"github.com/2bit-software/zombiekit/internal/server/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type WorkflowService struct {
	workflowv1connect.UnimplementedWorkflowServiceHandler
	storage storage.InitiativeStorage
}

func NewWorkflowService(storage storage.InitiativeStorage) *WorkflowService {
	return &WorkflowService{storage: storage}
}

func (s *WorkflowService) CreateInitiative(
	ctx context.Context,
	req *connect.Request[workflowv1.CreateInitiativeRequest],
) (*connect.Response[workflowv1.CreateInitiativeResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	if msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	init := &storage.Initiative{
		Name:        msg.Name,
		Type:        protoToInitType(msg.Type),
		Status:      storage.InitiativeStatusInProgress,
		Description: msg.Description,
		Steps:       defaultSteps(),
	}

	if err := s.storage.Create(ctx, init); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workflowv1.CreateInitiativeResponse{
		Initiative: initToProto(init),
	}), nil
}

func (s *WorkflowService) GetStatus(
	ctx context.Context,
	req *connect.Request[workflowv1.GetStatusRequest],
) (*connect.Response[workflowv1.GetStatusResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	if msg.InitiativeId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("initiative_id is required"))
	}

	id, err := uuid.Parse(msg.InitiativeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid initiative_id"))
	}

	init, err := s.storage.Get(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrInitiativeNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workflowv1.GetStatusResponse{
		Initiative: initToProto(init),
	}), nil
}

func (s *WorkflowService) UpdateStep(
	ctx context.Context,
	req *connect.Request[workflowv1.UpdateStepRequest],
) (*connect.Response[workflowv1.UpdateStepResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	if msg.InitiativeId == "" || msg.StepName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("initiative_id and step_name are required"))
	}

	init, err := s.getInitiative(ctx, msg.InitiativeId)
	if err != nil {
		return nil, err
	}

	if err := updateStepInPlace(init, msg.StepName, protoToStepStatus(msg.Status)); err != nil {
		return nil, err
	}

	if err := s.storage.Update(ctx, init); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workflowv1.UpdateStepResponse{
		Initiative: initToProto(init),
	}), nil
}

// getInitiative parses the ID and fetches the initiative, returning connect errors.
func (s *WorkflowService) getInitiative(ctx context.Context, rawID string) (*storage.Initiative, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid initiative_id"))
	}
	init, err := s.storage.Get(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrInitiativeNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return init, nil
}

// updateStepInPlace finds a step by name and updates its status in place.
func updateStepInPlace(init *storage.Initiative, stepName string, status storage.StepStatus) error {
	for i := range init.Steps {
		if init.Steps[i].Name == stepName {
			init.Steps[i].Status = status
			init.Steps[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return connect.NewError(connect.CodeNotFound, errors.New("step not found"))
}

func (s *WorkflowService) CompleteInitiative(
	ctx context.Context,
	req *connect.Request[workflowv1.CompleteInitiativeRequest],
) (*connect.Response[workflowv1.CompleteInitiativeResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	if msg.InitiativeId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("initiative_id is required"))
	}

	id, err := uuid.Parse(msg.InitiativeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid initiative_id"))
	}

	init, err := s.storage.Get(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrInitiativeNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	init.Status = storage.InitiativeStatusCompleted
	if err := s.storage.Update(ctx, init); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workflowv1.CompleteInitiativeResponse{
		Initiative: initToProto(init),
	}), nil
}

func (s *WorkflowService) ListInitiatives(
	ctx context.Context,
	req *connect.Request[workflowv1.ListInitiativesRequest],
) (*connect.Response[workflowv1.ListInitiativesResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	limit := 50
	if msg.Pagination != nil && msg.Pagination.PageSize > 0 {
		limit = int(msg.Pagination.PageSize)
	}

	var statusFilter *storage.InitiativeStatus
	if msg.StatusFilter != workflowv1.InitiativeStatus_INITIATIVE_STATUS_UNSPECIFIED {
		status := protoToInitStatus(msg.StatusFilter)
		statusFilter = &status
	}

	initiatives, err := s.storage.List(ctx, statusFilter, limit, 0)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &workflowv1.ListInitiativesResponse{
		Initiatives: make([]*workflowv1.Initiative, 0, len(initiatives)),
	}
	for _, init := range initiatives {
		resp.Initiatives = append(resp.Initiatives, initToProto(init))
	}

	return connect.NewResponse(resp), nil
}

func defaultSteps() []storage.WorkflowStep {
	return []storage.WorkflowStep{
		{Name: "spec", Status: storage.StepStatusPending},
		{Name: "plan", Status: storage.StepStatusPending},
		{Name: "tasks", Status: storage.StepStatusPending},
		{Name: "implement", Status: storage.StepStatusPending},
	}
}

func initToProto(init *storage.Initiative) *workflowv1.Initiative {
	steps := make([]*workflowv1.WorkflowStep, 0, len(init.Steps))
	for _, step := range init.Steps {
		steps = append(steps, &workflowv1.WorkflowStep{
			Name:      step.Name,
			Status:    stepStatusToProto(step.Status),
			UpdatedAt: timestamppb.New(step.UpdatedAt),
		})
	}

	return &workflowv1.Initiative{
		Id:          init.ID.String(),
		Name:        init.Name,
		Type:        initTypeToProto(init.Type),
		Status:      initStatusToProto(init.Status),
		Steps:       steps,
		CreatedAt:   timestamppb.New(init.CreatedAt),
		UpdatedAt:   timestamppb.New(init.UpdatedAt),
		Description: init.Description,
		BranchName:  init.BranchName,
	}
}

func initTypeToProto(t storage.InitiativeType) workflowv1.InitiativeType {
	switch t {
	case storage.InitiativeTypeFeature:
		return workflowv1.InitiativeType_INITIATIVE_TYPE_FEATURE
	case storage.InitiativeTypeBug:
		return workflowv1.InitiativeType_INITIATIVE_TYPE_BUG
	case storage.InitiativeTypeRefactor:
		return workflowv1.InitiativeType_INITIATIVE_TYPE_REFACTOR
	default:
		return workflowv1.InitiativeType_INITIATIVE_TYPE_UNSPECIFIED
	}
}

func protoToInitType(t workflowv1.InitiativeType) storage.InitiativeType {
	switch t {
	case workflowv1.InitiativeType_INITIATIVE_TYPE_FEATURE:
		return storage.InitiativeTypeFeature
	case workflowv1.InitiativeType_INITIATIVE_TYPE_BUG:
		return storage.InitiativeTypeBug
	case workflowv1.InitiativeType_INITIATIVE_TYPE_REFACTOR:
		return storage.InitiativeTypeRefactor
	default:
		return storage.InitiativeTypeFeature
	}
}

func initStatusToProto(s storage.InitiativeStatus) workflowv1.InitiativeStatus {
	switch s {
	case storage.InitiativeStatusInProgress:
		return workflowv1.InitiativeStatus_INITIATIVE_STATUS_IN_PROGRESS
	case storage.InitiativeStatusCompleted:
		return workflowv1.InitiativeStatus_INITIATIVE_STATUS_COMPLETED
	default:
		return workflowv1.InitiativeStatus_INITIATIVE_STATUS_UNSPECIFIED
	}
}

func protoToInitStatus(s workflowv1.InitiativeStatus) storage.InitiativeStatus {
	switch s {
	case workflowv1.InitiativeStatus_INITIATIVE_STATUS_IN_PROGRESS:
		return storage.InitiativeStatusInProgress
	case workflowv1.InitiativeStatus_INITIATIVE_STATUS_COMPLETED:
		return storage.InitiativeStatusCompleted
	default:
		return storage.InitiativeStatusInProgress
	}
}

func stepStatusToProto(s storage.StepStatus) workflowv1.StepStatus {
	switch s {
	case storage.StepStatusPending:
		return workflowv1.StepStatus_STEP_STATUS_PENDING
	case storage.StepStatusInProgress:
		return workflowv1.StepStatus_STEP_STATUS_IN_PROGRESS
	case storage.StepStatusCompleted:
		return workflowv1.StepStatus_STEP_STATUS_COMPLETED
	case storage.StepStatusSkipped:
		return workflowv1.StepStatus_STEP_STATUS_SKIPPED
	default:
		return workflowv1.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

func protoToStepStatus(s workflowv1.StepStatus) storage.StepStatus {
	switch s {
	case workflowv1.StepStatus_STEP_STATUS_PENDING:
		return storage.StepStatusPending
	case workflowv1.StepStatus_STEP_STATUS_IN_PROGRESS:
		return storage.StepStatusInProgress
	case workflowv1.StepStatus_STEP_STATUS_COMPLETED:
		return storage.StepStatusCompleted
	case workflowv1.StepStatus_STEP_STATUS_SKIPPED:
		return storage.StepStatusSkipped
	default:
		return storage.StepStatusPending
	}
}
