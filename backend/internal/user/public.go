package user

import (
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/ctxkey"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/gin-gonic/gin"
)

// AccountHandler — HTTP layer cho flow tài khoản công khai (/api/v1/public).
type AccountHandler struct {
	svc *AccountService
}

func NewAccountHandler(svc *AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

// RegisterPublicRoutes — group /api/v1/public/auth (không cần token).
func (h *AccountHandler) RegisterPublicRoutes(g *gin.RouterGroup) {
	g.POST("/register", h.register)
	g.POST("/verify-account", h.verifyAccount)
	g.POST("/resend-otp", h.resendOTP)
	g.POST("/forgot-password", h.forgotPassword)
	g.POST("/reset-password", h.resetPassword)
}

// RegisterProfileRoutes — group /api/v1/public (đã RequireAuth): tài khoản của chính mình.
func (h *AccountHandler) RegisterProfileRoutes(g *gin.RouterGroup) {
	g.GET("/me", h.me)
	g.PUT("/me", h.updateMe)
	g.POST("/me/avatar", h.uploadAvatar)
	g.POST("/me/change-password", h.changePassword)
}

type registerBody struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,min=2,max=120"`
}

func (h *AccountHandler) register(c *gin.Context) {
	var body registerBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" || len(body.Password) < 8 || strings.TrimSpace(body.FullName) == "" {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.register_required"))
		return
	}
	u, err := h.svc.Register(c, body.Email, body.Password, body.FullName)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{"user": toDTO(u), "message": "Vui lòng kiểm tra email để xác thực tài khoản"})
}

type otpBody struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func (h *AccountHandler) verifyAccount(c *gin.Context) {
	var body otpBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" || len(body.OTP) != 6 {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.email_otp_required"))
		return
	}
	if err := h.svc.VerifyAccount(c, body.Email, body.OTP); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Tài khoản đã được xác thực"})
}

type resendBody struct {
	Email string `json:"email"`
	Type  string `json:"type"` // verify-account | password-reset
}

func (h *AccountHandler) resendOTP(c *gin.Context) {
	var body resendBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.email_required"))
		return
	}
	switch body.Type {
	case "verify-account":
		if err := h.svc.ResendVerification(c, body.Email); err != nil {
			response.Error(c, err)
			return
		}
	case "password-reset":
		if err := h.svc.ForgotPassword(c, body.Email); err != nil {
			response.Error(c, err)
			return
		}
	default:
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_type", "error.invalid_type"))
		return
	}
	response.OK(c, gin.H{"message": "Đã gửi lại mã xác thực"})
}

func (h *AccountHandler) forgotPassword(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.email_required"))
		return
	}
	if err := h.svc.ForgotPassword(c, body.Email); err != nil {
		response.Error(c, err)
		return
	}
	// Luôn trả thành công (không lộ email tồn tại)
	response.OK(c, gin.H{"message": "Nếu email tồn tại, bạn sẽ nhận được mã đặt lại mật khẩu"})
}

type resetBody struct {
	Email       string `json:"email"`
	OTP         string `json:"otp"`
	NewPassword string `json:"new_password"`
}

func (h *AccountHandler) resetPassword(c *gin.Context) {
	var body resetBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" || len(body.OTP) != 6 || len(body.NewPassword) < 8 {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.reset_password_required"))
		return
	}
	if err := h.svc.ResetPassword(c, body.Email, body.OTP, body.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Mật khẩu đã được đặt lại, bạn có thể đăng nhập"})
}

// ---- profile (authed) ----

func (h *AccountHandler) me(c *gin.Context) {
	u, err := h.svc.users.GetByID(c, ctxkey.UserID(c))
	if err != nil {
		response.Error(c, apperr.WrapInternal(err))
		return
	}
	response.OK(c, toDTO(u))
}

func (h *AccountHandler) updateMe(c *gin.Context) {
	var body struct {
		FullName string `json:"full_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.FullName) == "" {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.full_name_required"))
		return
	}
	u, err := h.svc.UpdateProfile(c, ctxkey.UserID(c), strings.TrimSpace(body.FullName))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, toDTO(u))
}

func (h *AccountHandler) uploadAvatar(c *gin.Context) {
	fh, err := c.FormFile("avatar")
	if err != nil {
		response.Error(c, apperr.New(apperr.KindInvalid, "missing_avatar", "error.missing_avatar"))
		return
	}
	url, err := h.svc.UpdateAvatar(c, ctxkey.UserID(c), fh)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"avatar_url": url})
}

type changeMyPasswordBody struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AccountHandler) changePassword(c *gin.Context) {
	var body changeMyPasswordBody
	if err := c.ShouldBindJSON(&body); err != nil || body.CurrentPassword == "" || len(body.NewPassword) < 8 {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.change_password_required"))
		return
	}
	if err := h.svc.ChangePassword(c, ctxkey.UserID(c), body.CurrentPassword, body.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Mật khẩu đã được đổi"})
}

// toDTO — user → DTO công khai (không lộ password_hash).
func toDTO(u db.User) UserDTO {
	return UserDTO{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		AvatarURL: u.AvatarUrl,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}

// ToPublicDTO — export cho package khác (server/admin handler).
func ToPublicDTO(u db.User) UserDTO { return toDTO(u) }
