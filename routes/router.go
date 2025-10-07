package routes // Router setup layer.

import ( // Imports used in the router.
	"time" // For JWT expiration type.

	"HelmyTask/handlers" // User handler constructor.
	"HelmyTask/middlewares" // Logging & recovery & auth middlewares.
	"HelmyTask/services" // User service interface.

	"github.com/gin-gonic/gin" // Gin router.
)

// Setup attaches middlewares and registers all endpoints.
func Setup(r *gin.Engine, svc services.UserService, jwtSecret string, jwtExp time.Duration) {
	// Attach standard middlewares globally.
	r.Use(middlewares.RequestLogger(), middlewares.Recovery()) // Access log + panic recovery.

	// Swagger (if you have docs/swagger.yaml); serves static file at /swagger.yaml.
	r.StaticFile("/swagger.yaml", "./docs/swagger.yaml")

	// Group API under /api/v1 for versioning.
	api := r.Group("/api/v1")

	// Create the user handler (injecting service + JWT parameters).
	uh := handlers.NewUserHandler(svc, jwtSecret, jwtExp)

	// Public auth endpoints (no JWT required).
	api.POST("/auth/register", uh.Register) // Register new user.
	api.POST("/auth/login", uh.Login) // Login and get JWT.

	// Protected group (requires valid Authorization: Bearer <token>).
	protected := api.Group("/")
	protected.Use(middlewares.Auth(jwtSecret)) // JWT auth middleware.

	// "Me" endpoint (current user).
	protected.GET("/me", uh.GetUser) // You could point to a dedicated 'Me' handler; here we reuse GetUser with context in your baseline.

	// RESTful CRUD for users (admin-style).
	protected.POST("/users", uh.CreateUser) // Create
	protected.GET("/users", uh.ListUsers) // List (paginated)
	protected.GET("/users/:id", uh.GetUser) // Read (one)
	protected.PUT("/users/:id", uh.UpdateUser) // Update (partial)
	protected.DELETE("/users/:id", uh.DeleteUser) // Delete
}
