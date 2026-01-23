package models

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/memohai/memoh/internal/db/sqlc"
)

// Service provides CRUD operations for models
type Service struct {
	queries *sqlc.Queries
}

// NewService creates a new models service
func NewService(queries *sqlc.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

// Create adds a new model to the database
func (s *Service) Create(ctx context.Context, req AddRequest) (AddResponse, error) {
	model := Model(req)
	if err := model.Validate(); err != nil {
		return AddResponse{}, fmt.Errorf("validation failed: %w", err)
	}

	// Convert to sqlc params
	params := sqlc.CreateModelParams{
		ModelID:    model.ModelID,
		BaseUrl:    model.BaseURL,
		ApiKey:     model.APIKey,
		ClientType: string(model.ClientType),
		Type:       string(model.Type),
	}

	// Handle optional name field
	if model.Name != "" {
		params.Name = pgtype.Text{String: model.Name, Valid: true}
	}

	// Handle optional dimensions field (only for embedding models)
	if model.Type == ModelTypeEmbedding && model.Dimensions > 0 {
		params.Dimensions = pgtype.Int4{Int32: int32(model.Dimensions), Valid: true}
	}

	created, err := s.queries.CreateModel(ctx, params)
	if err != nil {
		return AddResponse{}, fmt.Errorf("failed to create model: %w", err)
	}

	// Convert pgtype.UUID to string
	var idStr string
	if created.ID.Valid {
		id, err := uuid.FromBytes(created.ID.Bytes[:])
		if err != nil {
			return AddResponse{}, fmt.Errorf("failed to convert UUID: %w", err)
		}
		idStr = id.String()
	}

	return AddResponse{
		ID:      idStr,
		ModelID: created.ModelID,
	}, nil
}

// GetByID retrieves a model by its internal UUID
func (s *Service) GetByID(ctx context.Context, id string) (GetResponse, error) {
	uuid, err := parseUUID(id)
	if err != nil {
		return GetResponse{}, fmt.Errorf("invalid ID: %w", err)
	}

	dbModel, err := s.queries.GetModelByID(ctx, uuid)
	if err != nil {
		return GetResponse{}, fmt.Errorf("failed to get model: %w", err)
	}

	return convertToGetResponse(dbModel), nil
}

// GetByModelID retrieves a model by its model_id field
func (s *Service) GetByModelID(ctx context.Context, modelID string) (GetResponse, error) {
	if modelID == "" {
		return GetResponse{}, fmt.Errorf("model_id is required")
	}

	dbModel, err := s.queries.GetModelByModelID(ctx, modelID)
	if err != nil {
		return GetResponse{}, fmt.Errorf("failed to get model: %w", err)
	}

	return convertToGetResponse(dbModel), nil
}

// List returns all models
func (s *Service) List(ctx context.Context) ([]GetResponse, error) {
	dbModels, err := s.queries.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	return convertToGetResponseList(dbModels), nil
}

// ListByType returns models filtered by type (chat or embedding)
func (s *Service) ListByType(ctx context.Context, modelType ModelType) ([]GetResponse, error) {
	if modelType != ModelTypeChat && modelType != ModelTypeEmbedding {
		return nil, fmt.Errorf("invalid model type: %s", modelType)
	}

	dbModels, err := s.queries.ListModelsByType(ctx, string(modelType))
	if err != nil {
		return nil, fmt.Errorf("failed to list models by type: %w", err)
	}

	return convertToGetResponseList(dbModels), nil
}

// ListByClientType returns models filtered by client type
func (s *Service) ListByClientType(ctx context.Context, clientType ClientType) ([]GetResponse, error) {
	if clientType != ClientTypeOpenAI && clientType != ClientTypeAnthropic && clientType != ClientTypeGoogle {
		return nil, fmt.Errorf("invalid client type: %s", clientType)
	}

	dbModels, err := s.queries.ListModelsByClientType(ctx, string(clientType))
	if err != nil {
		return nil, fmt.Errorf("failed to list models by client type: %w", err)
	}

	return convertToGetResponseList(dbModels), nil
}

// UpdateByID updates a model by its internal UUID
func (s *Service) UpdateByID(ctx context.Context, id string, req UpdateRequest) (GetResponse, error) {
	uuid, err := parseUUID(id)
	if err != nil {
		return GetResponse{}, fmt.Errorf("invalid ID: %w", err)
	}

	model := Model(req)
	if err := model.Validate(); err != nil {
		return GetResponse{}, fmt.Errorf("validation failed: %w", err)
	}

	params := sqlc.UpdateModelParams{
		ID:         uuid,
		BaseUrl:    model.BaseURL,
		ApiKey:     model.APIKey,
		ClientType: string(model.ClientType),
		Type:       string(model.Type),
	}

	if model.Name != "" {
		params.Name = pgtype.Text{String: model.Name, Valid: true}
	}

	if model.Type == ModelTypeEmbedding && model.Dimensions > 0 {
		params.Dimensions = pgtype.Int4{Int32: int32(model.Dimensions), Valid: true}
	}

	updated, err := s.queries.UpdateModel(ctx, params)
	if err != nil {
		return GetResponse{}, fmt.Errorf("failed to update model: %w", err)
	}

	return convertToGetResponse(updated), nil
}

// UpdateByModelID updates a model by its model_id field
func (s *Service) UpdateByModelID(ctx context.Context, modelID string, req UpdateRequest) (GetResponse, error) {
	if modelID == "" {
		return GetResponse{}, fmt.Errorf("model_id is required")
	}

	model := Model(req)
	if err := model.Validate(); err != nil {
		return GetResponse{}, fmt.Errorf("validation failed: %w", err)
	}

	params := sqlc.UpdateModelByModelIDParams{
		ModelID:    modelID,
		BaseUrl:    model.BaseURL,
		ApiKey:     model.APIKey,
		ClientType: string(model.ClientType),
		Type:       string(model.Type),
	}

	if model.Name != "" {
		params.Name = pgtype.Text{String: model.Name, Valid: true}
	}

	if model.Type == ModelTypeEmbedding && model.Dimensions > 0 {
		params.Dimensions = pgtype.Int4{Int32: int32(model.Dimensions), Valid: true}
	}

	updated, err := s.queries.UpdateModelByModelID(ctx, params)
	if err != nil {
		return GetResponse{}, fmt.Errorf("failed to update model: %w", err)
	}

	return convertToGetResponse(updated), nil
}

// DeleteByID deletes a model by its internal UUID
func (s *Service) DeleteByID(ctx context.Context, id string) error {
	uuid, err := parseUUID(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	if err := s.queries.DeleteModel(ctx, uuid); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}

// DeleteByModelID deletes a model by its model_id field
func (s *Service) DeleteByModelID(ctx context.Context, modelID string) error {
	if modelID == "" {
		return fmt.Errorf("model_id is required")
	}

	if err := s.queries.DeleteModelByModelID(ctx, modelID); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}

// Count returns the total number of models
func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.queries.CountModels(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count models: %w", err)
	}
	return count, nil
}

// CountByType returns the number of models of a specific type
func (s *Service) CountByType(ctx context.Context, modelType ModelType) (int64, error) {
	if modelType != ModelTypeChat && modelType != ModelTypeEmbedding {
		return 0, fmt.Errorf("invalid model type: %s", modelType)
	}

	count, err := s.queries.CountModelsByType(ctx, string(modelType))
	if err != nil {
		return 0, fmt.Errorf("failed to count models by type: %w", err)
	}
	return count, nil
}

// Helper functions

func parseUUID(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID format: %w", err)
	}

	var pgUUID pgtype.UUID
	copy(pgUUID.Bytes[:], parsed[:])
	pgUUID.Valid = true

	return pgUUID, nil
}

func convertToGetResponse(dbModel sqlc.Model) GetResponse {
	resp := GetResponse{
		ModelId: dbModel.ModelID,
		Model: Model{
			ModelID:    dbModel.ModelID,
			BaseURL:    dbModel.BaseUrl,
			APIKey:     dbModel.ApiKey,
			ClientType: ClientType(dbModel.ClientType),
			Type:       ModelType(dbModel.Type),
		},
	}

	if dbModel.Name.Valid {
		resp.Model.Name = dbModel.Name.String
	}

	if dbModel.Dimensions.Valid {
		resp.Model.Dimensions = int(dbModel.Dimensions.Int32)
	}

	return resp
}

func convertToGetResponseList(dbModels []sqlc.Model) []GetResponse {
	responses := make([]GetResponse, 0, len(dbModels))
	for _, dbModel := range dbModels {
		responses = append(responses, convertToGetResponse(dbModel))
	}
	return responses
}
