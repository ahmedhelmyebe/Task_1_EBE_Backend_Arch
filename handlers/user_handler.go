package handlers // Controller layer translates HTTP <-> service calls.

import ( // Imports needed by handlers.
	"net/http" // Status codes and HTTP primitives.
	"strconv" // String->int parsing for URL params.
	"time" // For passing JWT expiration to service login.

	"HelmyTask/models" // Request/response DTOs.
	"HelmyTask/services" // Use-case interface.

	"github.com/gin-gonic/gin" // Gin web framework.
)

// UserHandler bundles dependencies needed by user endpoints.
type UserHandler struct {
	svc        services.UserService // Injected business logic.
	jwtSecret  string // JWT signing secret configured in main.
	jwtExpires time.Duration // JWT validity duration.
}

// NewUserHandler constructs a handler for users with its dependencies.
func NewUserHandler(svc services.UserService, jwtSecret string, jwtExp time.Duration) *UserHandler {
	return &UserHandler{svc: svc, jwtSecret: jwtSecret, jwtExpires: jwtExp} // Return pointer for methods.
}

// Register handles POST /auth/register (public).
func (h *UserHandler) Register(c *gin.Context) {
	var req models.RegisterRequest // Allocate request payload struct.
	if err := c.ShouldBindJSON(&req); err != nil { // Bind and validate JSON input.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // 400 if validation fails.
		return // Stop handler here.
	}
	u, err := h.svc.Register(req) // Delegate to service (hash + save + optional cache warm).
	if err != nil { // Typically "email already exists".
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Report error to client.
		return
	}
	c.JSON(http.StatusCreated, u) // 201 Created with user JSON.
}

// Login handles POST /auth/login (public).
func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest // Allocate request payload struct.
	if err := c.ShouldBindJSON(&req); err != nil { // Bind/validate JSON.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // 400 on invalid input.
		return
	}
	tok, err := h.svc.Login(req, h.jwtSecret, h.jwtExpires) // Delegate to service (validates + signs JWT).
	if err != nil { // Wrong credentials → 401 Unauthorized.
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.AuthResponse{Token: tok}) // Return {"token": "..."}.
}

// GetUser handles GET /users/:id (protected).
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := parseUint(c.Param("id")) // Parse :id from URL.
	if err != nil { // Invalid ID → 400.
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	u, err := h.svc.GetUser(id) // Fetch user (cache-aware).
	if err != nil { // Not found → 404.
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, u) // Respond with user JSON.
}

// CreateUser handles POST /users (protected; typically admin-only).
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.RegisterRequest // Reuse register DTO (requires password).
	if err := c.ShouldBindJSON(&req); err != nil { // Bind/validate JSON.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.svc.CreateUser(req) // Service creates user (hash + uniqueness).
	if err != nil { // Business error → 400.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u) // 201 Created with user JSON.
}

// UpdateUser handles PUT /users/:id (protected).
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := parseUint(c.Param("id")) // Parse :id path param.
	if err != nil { // Invalid ID → 400.
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req models.UpdateUserRequest // Allocate partial-update DTO.
	if err := c.ShouldBindJSON(&req); err != nil { // Bind JSON; all fields optional.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.svc.UpdateUser(id, req) // Update via service (hash if password; refresh cache).
	if err != nil { // Could be "email exists" or not found.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u) // 200 OK with updated user.
}

// DeleteUser handles DELETE /users/:id (protected).
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := parseUint(c.Param("id")) // Parse :id.
	if err != nil { // Invalid ID → 400.
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteUser(id); err != nil { // Service delete (also clears cache).
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"}) // Simplified mapping to 404.
		return
	}
	c.Status(http.StatusNoContent) // 204 No Content on success (typical REST delete).
}

// ListUsers handles GET /users?page=1&limit=10 (protected).
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Parse query params with defaults; we let the service clamp them too.
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1")) // Default page=1.
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10")) // Default limit=10.

	paged, err := h.svc.ListUsers(page, limit) // Get page via service (items + total + page + limit).
	if err != nil { // Internal error → 500.
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, paged) // 200 OK with envelope.
}

// parseUint safely converts a numeric string to uint.
func parseUint(s string) (uint, error) {
	id64, err := strconv.ParseUint(s, 10, 0) // Parse base-10 as unsigned.
	return uint(id64), err // Cast to uint; return parse error if any.
}
