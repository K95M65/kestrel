package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.autonomous.ai/os/system/device"
	"go.autonomous.ai/os/system/server/serializers"
)

// GetHousehold is GET /api/device/household.
func (h *DeviceHandler) GetHousehold(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.GetHouseholdPublic(true)))
}

// GetClaimPublic is GET /api/device/claim — LAN, no admin. PIN only if unclaimed.
func (h *DeviceHandler) GetClaimPublic(c *gin.Context) {
	pub := h.service.GetHouseholdPublic(true)
	if pub.Claimed {
		pub.SetupPIN = ""
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(pub))
}

// ConfirmClaim is POST /api/device/claim.
func (h *DeviceHandler) ConfirmClaim(c *gin.Context) {
	var req device.ClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	pub, err := h.service.Claim(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(pub))
}

type inviteBody struct {
	Role string `json:"role"`
}

// StartInvite is POST /api/device/household/invite.
func (h *DeviceHandler) StartInvite(c *gin.Context) {
	var body inviteBody
	_ = c.ShouldBindJSON(&body)
	pub, err := h.service.StartInvite(body.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(pub))
}

type memberRoleBody struct {
	Role string `json:"role"`
}

// SetMemberRole is PUT /api/device/household/members/:label/role.
func (h *DeviceHandler) SetMemberRole(c *gin.Context) {
	label := strings.TrimSpace(c.Param("label"))
	var body memberRoleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.SetMemberRole(label, body.Role); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.GetHouseholdPublic(false)))
}

type roomBody struct {
	Room string `json:"room"`
}

// SetHouseholdRoom is PUT /api/device/household/room.
func (h *DeviceHandler) SetHouseholdRoom(c *gin.Context) {
	var body roomBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.SetHouseholdRoom(body.Room); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.GetHouseholdPublic(false)))
}
