package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/auth"
	"github.com/memohai/memoh/internal/bots"
	"github.com/memohai/memoh/internal/contacts"
	"github.com/memohai/memoh/internal/identity"
	"github.com/memohai/memoh/internal/users"
)

type ContactsHandler struct {
	service     *contacts.Service
	botService  *bots.Service
	userService *users.Service
}

func NewContactsHandler(service *contacts.Service, botService *bots.Service, userService *users.Service) *ContactsHandler {
	return &ContactsHandler{
		service:     service,
		botService:  botService,
		userService: userService,
	}
}

func (h *ContactsHandler) Register(e *echo.Echo) {
	group := e.Group("/bots/:bot_id/contacts")
	group.GET("", h.List)
	group.GET("/:id", h.Get)
	group.POST("", h.Create)
	group.PATCH("/:id", h.Update)
	group.POST("/:id/bind", h.Bind)
	group.POST("/:id/bind_token", h.IssueBindToken)
	group.POST("/bind_confirm", h.ConfirmBind)
}

type contactBindRequest struct {
	Platform   string `json:"platform"`
	ExternalID string `json:"external_id"`
	BindToken  string `json:"bind_token"`
}

type contactBindTokenRequest struct {
	TargetPlatform   string `json:"target_platform"`
	TargetExternalID string `json:"target_external_id"`
	TTLSeconds       int    `json:"ttl_seconds"`
}

type contactBindConfirmRequest struct {
	Token string `json:"token"`
}

func (h *ContactsHandler) List(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	query := strings.TrimSpace(c.QueryParam("q"))
	items, err := h.service.Search(c.Request().Context(), botID, query)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"items": items})
}

func (h *ContactsHandler) Get(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "contact id is required")
	}
	item, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, item)
}

func (h *ContactsHandler) Create(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	var req contacts.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.BotID = botID
	item, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, item)
}

func (h *ContactsHandler) Update(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "contact id is required")
	}
	var req contacts.UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	item, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, item)
}

func (h *ContactsHandler) Bind(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "contact id is required")
	}
	var req contactBindRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(req.BindToken) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bind_token is required")
	}
	token, err := h.service.GetBindToken(c.Request().Context(), req.BindToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid bind token")
	}
	if token.UsedAt.IsZero() == false {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token already used")
	}
	if time.Now().UTC().After(token.ExpiresAt) {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token expired")
	}
	if token.BotID != botID || token.ContactID != id {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token mismatch")
	}
	platform := strings.TrimSpace(req.Platform)
	externalID := strings.TrimSpace(req.ExternalID)
	if platform == "" || externalID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "platform and external_id are required")
	}
	if token.TargetPlatform != "" && token.TargetPlatform != platform {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token platform mismatch")
	}
	if token.TargetExternalID != "" && token.TargetExternalID != externalID {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token external_id mismatch")
	}
	bound, err := h.service.UpsertChannel(c.Request().Context(), botID, id, platform, externalID, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	_, _ = h.service.MarkBindTokenUsed(c.Request().Context(), token.ID)
	return c.JSON(http.StatusOK, bound)
}

func (h *ContactsHandler) IssueBindToken(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "contact id is required")
	}
	var req contactBindTokenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	ttl := 10 * time.Minute
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}
	token, err := h.service.CreateBindToken(c.Request().Context(), botID, id, req.TargetPlatform, req.TargetExternalID, userID, ttl)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, token)
}

func (h *ContactsHandler) ConfirmBind(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), userID, botID); err != nil {
		return err
	}
	var req contactBindConfirmRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	token, err := h.service.GetBindToken(c.Request().Context(), req.Token)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid bind token")
	}
	if token.UsedAt.IsZero() == false {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token already used")
	}
	if time.Now().UTC().After(token.ExpiresAt) {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token expired")
	}
	if token.BotID != botID {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token mismatch")
	}
	if token.IssuedByUserID != "" && token.IssuedByUserID != userID {
		return echo.NewHTTPError(http.StatusBadRequest, "bind token not issued for current user")
	}
	contact, err := h.service.GetByID(c.Request().Context(), token.ContactID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if contact.UserID != "" && contact.UserID != userID {
		return echo.NewHTTPError(http.StatusBadRequest, "contact already bound to another user")
	}
	if contact.UserID == "" {
		if _, err := h.service.BindUser(c.Request().Context(), contact.ID, userID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
	_, _ = h.service.MarkBindTokenUsed(c.Request().Context(), token.ID)
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ContactsHandler) requireUserID(c echo.Context) (string, error) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		return "", err
	}
	if err := identity.ValidateUserID(userID); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return userID, nil
}

func (h *ContactsHandler) authorizeBotAccess(ctx context.Context, actorID, botID string) (bots.Bot, error) {
	if h.botService == nil || h.userService == nil {
		return bots.Bot{}, echo.NewHTTPError(http.StatusInternalServerError, "bot services not configured")
	}
	isAdmin, err := h.userService.IsAdmin(ctx, actorID)
	if err != nil {
		return bots.Bot{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	bot, err := h.botService.AuthorizeAccess(ctx, actorID, botID, isAdmin, bots.AccessPolicy{AllowPublicMember: false})
	if err != nil {
		if errors.Is(err, bots.ErrBotNotFound) {
			return bots.Bot{}, echo.NewHTTPError(http.StatusNotFound, "bot not found")
		}
		if errors.Is(err, bots.ErrBotAccessDenied) {
			return bots.Bot{}, echo.NewHTTPError(http.StatusForbidden, "bot access denied")
		}
		return bots.Bot{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return bot, nil
}
