package settings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db/sqlc"
)

type Service struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewService(log *slog.Logger, queries *sqlc.Queries) *Service {
	return &Service{
		queries: queries,
		logger:  log.With(slog.String("service", "settings")),
	}
}

func (s *Service) Get(ctx context.Context, userID string) (Settings, error) {
	pgID, err := parseUUID(userID)
	if err != nil {
		return Settings{}, err
	}
	row, err := s.queries.GetSettingsByUserID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Settings{
				ChatModelID:        "",
				MemoryModelID:      "",
				EmbeddingModelID:   "",
				MaxContextLoadTime: DefaultMaxContextLoadTime,
				Language:           DefaultLanguage,
			}, nil
		}
		return Settings{}, err
	}
	return normalizeUserSetting(row), nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertRequest) (Settings, error) {
	if s.queries == nil {
		return Settings{}, fmt.Errorf("settings queries not configured")
	}
	pgID, err := parseUUID(userID)
	if err != nil {
		return Settings{}, err
	}

	current := Settings{
		ChatModelID:        "",
		MemoryModelID:      "",
		EmbeddingModelID:   "",
		MaxContextLoadTime: DefaultMaxContextLoadTime,
		Language:           DefaultLanguage,
	}
	existing, err := s.queries.GetSettingsByUserID(ctx, pgID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Settings{}, err
	}
	if err == nil {
		current = normalizeUserSetting(existing)
	}

	if value := strings.TrimSpace(req.ChatModelID); value != "" {
		current.ChatModelID = value
	}
	if value := strings.TrimSpace(req.MemoryModelID); value != "" {
		current.MemoryModelID = value
	}
	if value := strings.TrimSpace(req.EmbeddingModelID); value != "" {
		current.EmbeddingModelID = value
	}
	if req.MaxContextLoadTime != nil && *req.MaxContextLoadTime > 0 {
		current.MaxContextLoadTime = *req.MaxContextLoadTime
	}
	if strings.TrimSpace(req.Language) != "" {
		current.Language = strings.TrimSpace(req.Language)
	}

	_, err = s.queries.UpsertUserSettings(ctx, sqlc.UpsertUserSettingsParams{
		UserID:             pgID,
		ChatModelID:        pgtype.Text{String: current.ChatModelID, Valid: current.ChatModelID != ""},
		MemoryModelID:      pgtype.Text{String: current.MemoryModelID, Valid: current.MemoryModelID != ""},
		EmbeddingModelID:   pgtype.Text{String: current.EmbeddingModelID, Valid: current.EmbeddingModelID != ""},
		MaxContextLoadTime: int32(current.MaxContextLoadTime),
		Language:           current.Language,
	})
	if err != nil {
		return Settings{}, err
	}
	return current, nil
}

func (s *Service) GetBot(ctx context.Context, botID string) (Settings, error) {
	pgID, err := parseUUID(botID)
	if err != nil {
		return Settings{}, err
	}
	row, err := s.queries.GetSettingsByBotID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Settings{
				MaxContextLoadTime: DefaultMaxContextLoadTime,
				Language:           DefaultLanguage,
				AllowGuest:         false,
			}, nil
		}
		return Settings{}, err
	}
	return normalizeBotSetting(row), nil
}

func (s *Service) UpsertBot(ctx context.Context, botID string, req UpsertRequest) (Settings, error) {
	if s.queries == nil {
		return Settings{}, fmt.Errorf("settings queries not configured")
	}
	pgID, err := parseUUID(botID)
	if err != nil {
		return Settings{}, err
	}

	current := Settings{
		MaxContextLoadTime: DefaultMaxContextLoadTime,
		Language:           DefaultLanguage,
		AllowGuest:         false,
	}
	existing, err := s.queries.GetSettingsByBotID(ctx, pgID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Settings{}, err
	}
	if err == nil {
		current = normalizeBotSetting(existing)
	}
	if req.MaxContextLoadTime != nil && *req.MaxContextLoadTime > 0 {
		current.MaxContextLoadTime = *req.MaxContextLoadTime
	}
	if strings.TrimSpace(req.Language) != "" {
		current.Language = strings.TrimSpace(req.Language)
	}
	if req.AllowGuest != nil {
		current.AllowGuest = *req.AllowGuest
	}

	_, err = s.queries.UpsertBotSettings(ctx, sqlc.UpsertBotSettingsParams{
		BotID:              pgID,
		MaxContextLoadTime: int32(current.MaxContextLoadTime),
		Language:           current.Language,
		AllowGuest:         current.AllowGuest,
	})
	if err != nil {
		return Settings{}, err
	}
	return current, nil
}

func (s *Service) Delete(ctx context.Context, botID string) error {
	if s.queries == nil {
		return fmt.Errorf("settings queries not configured")
	}
	pgID, err := parseUUID(botID)
	if err != nil {
		return err
	}
	return s.queries.DeleteSettingsByBotID(ctx, pgID)
}

func normalizeUserSetting(row sqlc.UserSetting) Settings {
	settings := Settings{
		ChatModelID:        strings.TrimSpace(row.ChatModelID.String),
		MemoryModelID:      strings.TrimSpace(row.MemoryModelID.String),
		EmbeddingModelID:   strings.TrimSpace(row.EmbeddingModelID.String),
		MaxContextLoadTime: int(row.MaxContextLoadTime),
		Language:           strings.TrimSpace(row.Language),
	}
	if settings.MaxContextLoadTime <= 0 {
		settings.MaxContextLoadTime = DefaultMaxContextLoadTime
	}
	if settings.Language == "" {
		settings.Language = DefaultLanguage
	}
	return settings
}

func normalizeBotSetting(row sqlc.BotSetting) Settings {
	settings := Settings{
		MaxContextLoadTime: int(row.MaxContextLoadTime),
		Language:           strings.TrimSpace(row.Language),
		AllowGuest:         row.AllowGuest,
	}
	if settings.MaxContextLoadTime <= 0 {
		settings.MaxContextLoadTime = DefaultMaxContextLoadTime
	}
	if settings.Language == "" {
		settings.Language = DefaultLanguage
	}
	return settings
}

func parseUUID(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	var pgID pgtype.UUID
	pgID.Valid = true
	copy(pgID.Bytes[:], parsed[:])
	return pgID, nil
}
