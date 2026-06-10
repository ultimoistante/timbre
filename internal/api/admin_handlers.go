package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
)

func init() {
	// Admin routes are registered in registerRoutes via the admin group.
}

// registerAdminRoutes wires admin endpoints onto the admin group. Called from
// registerRoutes after the admin group is created.
func (s *Server) registerAdminRoutes(admin *echo.Group) {
	admin.GET("/users", s.handleAdminListUsers)
	admin.POST("/users", s.handleAdminCreateUser)
	admin.GET("/users/:id", s.handleAdminGetUser)
	admin.PATCH("/users/:id", s.handleAdminUpdateUser)
	admin.DELETE("/users/:id", s.handleAdminDeleteUser)
}

func (s *Server) handleAdminListUsers(c echo.Context) error {
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, users)
}

type createUserBody struct {
	Username   string      `json:"username"`
	Password   string      `json:"password"`
	Role       models.Role `json:"role"`
	QuotaBytes int64       `json:"quotaBytes"`
}

func (s *Server) handleAdminCreateUser(c echo.Context) error {
	var b createUserBody
	if err := c.Bind(&b); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if b.Username == "" || b.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username and password required")
	}
	if b.Role == "" {
		b.Role = models.RoleUser
	}

	hash, err := auth.HashPassword(b.Password)
	if err != nil {
		return err
	}

	user := models.User{
		Username:     b.Username,
		PasswordHash: hash,
		Role:         b.Role,
		QuotaBytes:   b.QuotaBytes,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusConflict, "username already taken")
	}
	if err := s.ensureUserRoot(user.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, user)
}

func (s *Server) handleAdminGetUser(c echo.Context) error {
	user, err := s.userByParam(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

type updateUserBody struct {
	Role       *models.Role `json:"role"`
	QuotaBytes *int64       `json:"quotaBytes"`
	Password   string       `json:"password"`
}

func (s *Server) handleAdminUpdateUser(c echo.Context) error {
	user, err := s.userByParam(c)
	if err != nil {
		return err
	}

	var b updateUserBody
	if err := c.Bind(&b); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	updates := map[string]any{}
	if b.Role != nil {
		updates["role"] = *b.Role
	}
	if b.QuotaBytes != nil {
		updates["quota_bytes"] = *b.QuotaBytes
	}
	if b.Password != "" {
		h, err := auth.HashPassword(b.Password)
		if err != nil {
			return err
		}
		updates["password_hash"] = h
	}

	if len(updates) > 0 {
		if err := s.db.Model(user).Updates(updates).Error; err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, user)
}

func (s *Server) handleAdminDeleteUser(c echo.Context) error {
	user, err := s.userByParam(c)
	if err != nil {
		return err
	}
	// Prevent deleting the calling admin.
	if caller := auth.CurrentUser(c); caller.ID == user.ID {
		return echo.NewHTTPError(http.StatusConflict, "cannot delete yourself")
	}

	if err := s.db.Delete(user).Error; err != nil {
		return err
	}

	// Remove user's media root on disk.
	root := filepath.Join(s.cfg.UsersDir(), strconv.FormatUint(uint64(user.ID), 10))
	_ = os.RemoveAll(root)

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) userByParam(c echo.Context) (*models.User, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return &user, nil
}
