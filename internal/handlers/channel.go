package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/auth"
	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/identity"
)

type ChannelHandler struct {
	service *channel.Service
}

func NewChannelHandler(service *channel.Service) *ChannelHandler {
	return &ChannelHandler{service: service}
}

func (h *ChannelHandler) Register(e *echo.Echo) {
	group := e.Group("/users/me/channels")
	group.GET("/:platform", h.GetUserConfig)
	group.PUT("/:platform", h.UpsertUserConfig)
}

// GetUserConfig godoc
// @Summary Get channel user config
// @Description Get channel binding configuration for current user
// @Tags channel
// @Param platform path string true "Channel platform"
// @Success 200 {object} channel.ChannelUserBinding
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/me/channels/{platform} [get]
func (h *ChannelHandler) GetUserConfig(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	channelType, err := channel.ParseChannelType(c.Param("platform"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	resp, err := h.service.GetUserConfig(c.Request().Context(), userID, channelType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

// UpsertUserConfig godoc
// @Summary Update channel user config
// @Description Update channel binding configuration for current user
// @Tags channel
// @Param platform path string true "Channel platform"
// @Param payload body channel.UpsertUserConfigRequest true "Channel user config payload"
// @Success 200 {object} channel.ChannelUserBinding
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/me/channels/{platform} [put]
func (h *ChannelHandler) UpsertUserConfig(c echo.Context) error {
	userID, err := h.requireUserID(c)
	if err != nil {
		return err
	}
	channelType, err := channel.ParseChannelType(c.Param("platform"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	var req channel.UpsertUserConfigRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Config == nil {
		req.Config = map[string]interface{}{}
	}
	resp, err := h.service.UpsertUserConfig(c.Request().Context(), userID, channelType, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *ChannelHandler) requireUserID(c echo.Context) (string, error) {
	userID, err := auth.UserIDFromContext(c)
	if err != nil {
		return "", err
	}
	if err := identity.ValidateUserID(userID); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return userID, nil
}
