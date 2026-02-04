package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db/sqlc"
)

type Service struct {
	queries *sqlc.Queries
}

func NewService(queries *sqlc.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) UpsertConfig(ctx context.Context, botID string, channelType ChannelType, req UpsertConfigRequest) (ChannelConfig, error) {
	if s.queries == nil {
		return ChannelConfig{}, fmt.Errorf("channel queries not configured")
	}
	if channelType == "" {
		return ChannelConfig{}, fmt.Errorf("channel type is required")
	}
	normalized, err := NormalizeChannelConfig(channelType, req.Credentials)
	if err != nil {
		return ChannelConfig{}, err
	}
	credentialsPayload, err := json.Marshal(normalized)
	if err != nil {
		return ChannelConfig{}, err
	}
	botUUID, err := parseUUID(botID)
	if err != nil {
		return ChannelConfig{}, err
	}
	selfIdentity := req.SelfIdentity
	if selfIdentity == nil {
		selfIdentity = map[string]interface{}{}
	}
	selfPayload, err := json.Marshal(selfIdentity)
	if err != nil {
		return ChannelConfig{}, err
	}
	routing := req.Routing
	if routing == nil {
		routing = map[string]interface{}{}
	}
	routingPayload, err := json.Marshal(routing)
	if err != nil {
		return ChannelConfig{}, err
	}
	capabilities := req.Capabilities
	if capabilities == nil {
		capabilities = map[string]interface{}{}
	}
	capabilitiesPayload, err := json.Marshal(capabilities)
	if err != nil {
		return ChannelConfig{}, err
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "pending"
	}
	verifiedAt := pgtype.Timestamptz{Valid: false}
	if req.VerifiedAt != nil {
		verifiedAt = pgtype.Timestamptz{Time: req.VerifiedAt.UTC(), Valid: true}
	}
	externalIdentity := strings.TrimSpace(req.ExternalIdentity)
	row, err := s.queries.UpsertBotChannelConfig(ctx, sqlc.UpsertBotChannelConfigParams{
		BotID:       botUUID,
		ChannelType: channelType.String(),
		Credentials: credentialsPayload,
		ExternalIdentity: pgtype.Text{
			String: externalIdentity,
			Valid:  externalIdentity != "",
		},
		SelfIdentity: selfPayload,
		Routing:      routingPayload,
		Capabilities: capabilitiesPayload,
		Status:       status,
		VerifiedAt:   verifiedAt,
	})
	if err != nil {
		return ChannelConfig{}, err
	}
	return normalizeChannelConfig(row)
}

func (s *Service) UpsertUserConfig(ctx context.Context, actorUserID string, channelType ChannelType, req UpsertUserConfigRequest) (ChannelUserBinding, error) {
	if s.queries == nil {
		return ChannelUserBinding{}, fmt.Errorf("channel queries not configured")
	}
	if channelType == "" {
		return ChannelUserBinding{}, fmt.Errorf("channel type is required")
	}
	normalized, err := NormalizeChannelUserConfig(channelType, req.Config)
	if err != nil {
		return ChannelUserBinding{}, err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return ChannelUserBinding{}, err
	}
	pgUserID, err := parseUUID(actorUserID)
	if err != nil {
		return ChannelUserBinding{}, err
	}
	row, err := s.queries.UpsertUserChannelBinding(ctx, sqlc.UpsertUserChannelBindingParams{
		UserID:      pgUserID,
		ChannelType: channelType.String(),
		Config:      payload,
	})
	if err != nil {
		return ChannelUserBinding{}, err
	}
	return normalizeChannelUserBindingRow(row)
}

func (s *Service) ResolveEffectiveConfig(ctx context.Context, botID string, channelType ChannelType) (ChannelConfig, error) {
	if s.queries == nil {
		return ChannelConfig{}, fmt.Errorf("channel queries not configured")
	}
	if channelType == "" {
		return ChannelConfig{}, fmt.Errorf("channel type is required")
	}
	if channelType == ChannelCLI || channelType == ChannelWeb {
		return ChannelConfig{
			ID:          channelType.String() + ":" + strings.TrimSpace(botID),
			BotID:       strings.TrimSpace(botID),
			ChannelType: channelType,
		}, nil
	}
	botUUID, err := parseUUID(botID)
	if err != nil {
		return ChannelConfig{}, err
	}
	row, err := s.queries.GetBotChannelConfig(ctx, sqlc.GetBotChannelConfigParams{
		BotID:       botUUID,
		ChannelType: channelType.String(),
	})
	if err == nil {
		return normalizeChannelConfig(row)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ChannelConfig{}, err
	}
	return ChannelConfig{}, fmt.Errorf("channel config not found")
}

func (s *Service) ListConfigsByType(ctx context.Context, channelType ChannelType) ([]ChannelConfig, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("channel queries not configured")
	}
	if channelType == ChannelCLI || channelType == ChannelWeb {
		return []ChannelConfig{}, nil
	}
	rows, err := s.queries.ListBotChannelConfigsByType(ctx, channelType.String())
	if err != nil {
		return nil, err
	}
	items := make([]ChannelConfig, 0, len(rows))
	for _, row := range rows {
		item, err := normalizeChannelConfig(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) GetUserConfig(ctx context.Context, actorUserID string, channelType ChannelType) (ChannelUserBinding, error) {
	if s.queries == nil {
		return ChannelUserBinding{}, fmt.Errorf("channel queries not configured")
	}
	if channelType == "" {
		return ChannelUserBinding{}, fmt.Errorf("channel type is required")
	}
	pgUserID, err := parseUUID(actorUserID)
	if err != nil {
		return ChannelUserBinding{}, err
	}
	row, err := s.queries.GetUserChannelBinding(ctx, sqlc.GetUserChannelBindingParams{
		UserID:      pgUserID,
		ChannelType: channelType.String(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChannelUserBinding{}, fmt.Errorf("channel user config not found")
		}
		return ChannelUserBinding{}, err
	}
	config, err := decodeConfigMap(row.Config)
	if err != nil {
		return ChannelUserBinding{}, err
	}
	return ChannelUserBinding{
		ID:          toUUIDString(row.ID),
		ChannelType: ChannelType(row.ChannelType),
		UserID:      toUUIDString(row.UserID),
		Config:      config,
		CreatedAt:   timeFromPg(row.CreatedAt),
		UpdatedAt:   timeFromPg(row.UpdatedAt),
	}, nil
}

func (s *Service) ListUserConfigsByType(ctx context.Context, channelType ChannelType) ([]ChannelUserBinding, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("channel queries not configured")
	}
	rows, err := s.queries.ListUserChannelBindingsByType(ctx, channelType.String())
	if err != nil {
		return nil, err
	}
	items := make([]ChannelUserBinding, 0, len(rows))
	for _, row := range rows {
		item, err := normalizeChannelUserBindingListRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) GetChannelSession(ctx context.Context, sessionID string) (ChannelSession, error) {
	if s.queries == nil {
		return ChannelSession{}, fmt.Errorf("channel queries not configured")
	}
	row, err := s.queries.GetChannelSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChannelSession{}, nil
		}
		return ChannelSession{}, err
	}
	return ChannelSession{
		SessionID:       row.SessionID,
		BotID:           toUUIDString(row.BotID),
		ChannelConfigID: toUUIDString(row.ChannelConfigID),
		UserID:          toUUIDString(row.UserID),
		ContactID:       toUUIDString(row.ContactID),
		Platform:        row.Platform,
		CreatedAt:       timeFromPg(row.CreatedAt),
		UpdatedAt:       timeFromPg(row.UpdatedAt),
	}, nil
}

func (s *Service) UpsertChannelSession(ctx context.Context, sessionID string, botID string, channelConfigID string, userID string, contactID string, platform string) error {
	if s.queries == nil {
		return fmt.Errorf("channel queries not configured")
	}
	pgUserID := pgtype.UUID{Valid: false}
	if strings.TrimSpace(userID) != "" {
		parsed, err := parseUUID(userID)
		if err != nil {
			return err
		}
		pgUserID = parsed
	}
	botUUID, err := parseUUID(botID)
	if err != nil {
		return err
	}
	var channelUUID pgtype.UUID
	if strings.TrimSpace(channelConfigID) != "" {
		channelUUID, err = parseUUID(channelConfigID)
		if err != nil {
			return err
		}
	}
	pgContactID := pgtype.UUID{Valid: false}
	if strings.TrimSpace(contactID) != "" {
		parsed, err := parseUUID(contactID)
		if err != nil {
			return err
		}
		pgContactID = parsed
	}
	_, err = s.queries.UpsertChannelSession(ctx, sqlc.UpsertChannelSessionParams{
		SessionID:       sessionID,
		BotID:           botUUID,
		ChannelConfigID: channelUUID,
		UserID:          pgUserID,
		ContactID:       pgContactID,
		Platform:        platform,
	})
	return err
}

func (s *Service) ResolveUserBinding(ctx context.Context, channelType ChannelType, criteria BindingCriteria) (string, error) {
	rows, err := s.ListUserConfigsByType(ctx, channelType)
	if err != nil {
		return "", err
	}
	switch channelType {
	case ChannelTelegram:
		for _, row := range rows {
			cfg, err := parseTelegramUserConfig(row.Config)
			if err != nil {
				continue
			}
			if matchTelegramBinding(cfg, criteria) {
				return row.UserID, nil
			}
		}
	case ChannelFeishu:
		for _, row := range rows {
			cfg, err := parseFeishuUserConfig(row.Config)
			if err != nil {
				continue
			}
			if matchFeishuBinding(cfg, criteria) {
				return row.UserID, nil
			}
		}
	default:
		return "", fmt.Errorf("unsupported channel type: %s", channelType)
	}
	return "", fmt.Errorf("channel user binding not found")
}

type BindingCriteria struct {
	Username string
	UserID   string
	ChatID   string
	OpenID   string
}

func matchTelegramBinding(cfg TelegramUserConfig, criteria BindingCriteria) bool {
	if criteria.ChatID != "" && cfg.ChatID != "" && criteria.ChatID == cfg.ChatID {
		return true
	}
	if criteria.UserID != "" && cfg.UserID != "" && criteria.UserID == cfg.UserID {
		return true
	}
	if criteria.Username != "" && cfg.Username != "" && strings.EqualFold(criteria.Username, cfg.Username) {
		return true
	}
	return false
}

func matchFeishuBinding(cfg FeishuUserConfig, criteria BindingCriteria) bool {
	if criteria.OpenID != "" && cfg.OpenID != "" && criteria.OpenID == cfg.OpenID {
		return true
	}
	if criteria.UserID != "" && cfg.UserID != "" && criteria.UserID == cfg.UserID {
		return true
	}
	return false
}

func normalizeChannelConfig(row sqlc.BotChannelConfig) (ChannelConfig, error) {
	credentials, err := decodeConfigMap(row.Credentials)
	if err != nil {
		return ChannelConfig{}, err
	}
	selfIdentity, err := decodeConfigMap(row.SelfIdentity)
	if err != nil {
		return ChannelConfig{}, err
	}
	routing, err := decodeConfigMap(row.Routing)
	if err != nil {
		return ChannelConfig{}, err
	}
	capabilities, err := decodeConfigMap(row.Capabilities)
	if err != nil {
		return ChannelConfig{}, err
	}
	verifiedAt := time.Time{}
	if row.VerifiedAt.Valid {
		verifiedAt = row.VerifiedAt.Time
	}
	externalIdentity := ""
	if row.ExternalIdentity.Valid {
		externalIdentity = strings.TrimSpace(row.ExternalIdentity.String)
	}
	return ChannelConfig{
		ID:               toUUIDString(row.ID),
		BotID:            toUUIDString(row.BotID),
		ChannelType:      ChannelType(row.ChannelType),
		Credentials:      credentials,
		ExternalIdentity: externalIdentity,
		SelfIdentity:     selfIdentity,
		Routing:          routing,
		Capabilities:     capabilities,
		Status:           strings.TrimSpace(row.Status),
		VerifiedAt:       verifiedAt,
		CreatedAt:        timeFromPg(row.CreatedAt),
		UpdatedAt:        timeFromPg(row.UpdatedAt),
	}, nil
}

func normalizeChannelUserBindingRow(row sqlc.UserChannelBinding) (ChannelUserBinding, error) {
	config, err := decodeConfigMap(row.Config)
	if err != nil {
		return ChannelUserBinding{}, err
	}
	return ChannelUserBinding{
		ID:          toUUIDString(row.ID),
		ChannelType: ChannelType(row.ChannelType),
		UserID:      toUUIDString(row.UserID),
		Config:      config,
		CreatedAt:   timeFromPg(row.CreatedAt),
		UpdatedAt:   timeFromPg(row.UpdatedAt),
	}, nil
}

func normalizeChannelUserBindingListRow(row sqlc.UserChannelBinding) (ChannelUserBinding, error) {
	config, err := decodeConfigMap(row.Config)
	if err != nil {
		return ChannelUserBinding{}, err
	}
	return ChannelUserBinding{
		ID:          toUUIDString(row.ID),
		ChannelType: ChannelType(row.ChannelType),
		UserID:      toUUIDString(row.UserID),
		Config:      config,
		CreatedAt:   timeFromPg(row.CreatedAt),
		UpdatedAt:   timeFromPg(row.UpdatedAt),
	}, nil
}

func parseUUID(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	var pgID pgtype.UUID
	pgID.Valid = true
	copy(pgID.Bytes[:], parsed[:])
	return pgID, nil
}

func toUUIDString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	parsed, err := uuid.FromBytes(value.Bytes[:])
	if err != nil {
		return ""
	}
	return parsed.String()
}

func timeFromPg(value pgtype.Timestamptz) time.Time {
	if value.Valid {
		return value.Time
	}
	return time.Time{}
}

func (c ChannelType) String() string {
	return string(c)
}
