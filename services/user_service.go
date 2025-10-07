package services // Use-case layer; orchestrates business rules, not HTTP/DB details.

import ( // Imports for this service layer.
	"context" // For Redis commands (need a Context).
	"encoding/json" // For caching user structs as JSON strings in Redis.
	"errors" // For returning friendly domain errors (e.g., "email already exists").
	"fmt" // For formatting Redis cache keys.
	"time" // For TTLs and JWT expiration.

	"HelmyTask/core" // Domain helpers; e.g., NormalizeName.
	"HelmyTask/models" // DTOs and User model.
	"HelmyTask/repositories" // Repository interface.
	"HelmyTask/utils" // HashPassword / CheckPassword helpers.
	"HelmyTask/utils/redislog" // Redis logger interface (your provided file).

	"github.com/golang-jwt/jwt/v5" // JWT token creation/signing.
	"github.com/redis/go-redis/v9" // Redis client for cache.
)

// UserService lists all use-cases that handlers can call.
type UserService interface {
	// Auth & read:
	Register(req models.RegisterRequest) (*models.User, error) // Public register.
	Login(req models.LoginRequest, jwtSecret string, exp time.Duration) (string, error) // Login and get JWT.
	GetByID(id uint) (*models.User, error) // Fetch one (cache-aware); used by /me.

	// CRUD:
	CreateUser(req models.RegisterRequest) (*models.User, error) // Admin create (same behavior as register).
	GetUser(id uint) (*models.User, error) // Read one; alias of GetByID for clarity.
	UpdateUser(id uint, req models.UpdateUserRequest) (*models.User, error) // Partial update.
	DeleteUser(id uint) error // Delete by ID.
	ListUsers(page, limit int) (*models.PagedUsers, error) // Paginated list.
}

// userService is the concrete implementation; it depends on repo + Redis + Redis logger.
type userService struct {
	repo repositories.UserRepository // Data access abstraction.
	rdb  *redis.Client // Redis client (may be nil if cache disabled).
	log  *redislog.Logger // Redis logger (may be nil if not configured).
}

// NewUserService constructs a service with all dependencies injected.
func NewUserService(repo repositories.UserRepository, rdb *redis.Client, rlog *redislog.Logger) UserService {
	return &userService{repo: repo, rdb: rdb, log: rlog} // Return a struct implementing the interface.
}

// userCacheTTL is how long a cached user stays in Redis before expiring.
const userCacheTTL = 10 * time.Minute // Adjust based on your read/write pattern.

// cacheKeyUser formats a consistent Redis key for a user's cached JSON.
func (s *userService) cacheKeyUser(id uint) string {
	return fmt.Sprintf("user:%d", id) // e.g., "user:42".
}

// ---------------- Auth & single read ----------------

// Register creates a new user (after checking email uniqueness), hashes password, and warms cache.
func (s *userService) Register(req models.RegisterRequest) (*models.User, error) {
	// Check for existing email to maintain uniqueness.
	if _, err := s.repo.FindByEmail(req.Email); err == nil { // If no error, a row with that email exists.
		if s.log != nil { s.log.Warn("register email exists", map[string]string{"email": req.Email}) } // Log to Redis.
		return nil, errors.New("email already exists") // Return a friendly message for the handler.
	}

	// Hash the incoming plaintext password before saving.
	hash, err := utils.HashPassword(req.Password) // Uses bcrypt or similar; defined in utils.
	if err != nil { // If hashing fails, log and return error.
		if s.log != nil { s.log.Error("register hash error", map[string]string{"email": req.Email, "err": err.Error()}) }
		return nil, err
	}

	// Build the new User entity (domain-normalized name).
	u := &models.User{
		Name:     core.NormalizeName(req.Name), // Apply any naming rules (e.g., capitalize).
		Email:    req.Email, // Store unique email.
		Password: hash, // Store hashed password, not plaintext.
	}

	// Insert into the database.
	if err := s.repo.Create(u); err != nil { // Will set u.ID on success.
		if s.log != nil { s.log.Error("register db create error", map[string]string{"email": req.Email, "err": err.Error()}) }
		return nil, err
	}

	// Optionally warm cache: write the JSON into Redis so the first /me is a HIT.
	if s.rdb != nil { // Only if Redis is configured.
		ctx := context.Background() // Use a background context for one-off calls.
		if b, _ := json.Marshal(u); len(b) > 0 { // Marshal struct -> JSON bytes.
			_ = s.rdb.Set(ctx, s.cacheKeyUser(u.ID), b, userCacheTTL).Err() // SET key value EX ttl
			if s.log != nil { s.log.Info("cache warm after register", map[string]string{"key": s.cacheKeyUser(u.ID), "user_id": fmt.Sprint(u.ID)}) }
		}
	}

	// Log final success of the registration flow.
	if s.log != nil { s.log.Info("register success", map[string]string{"user_id": fmt.Sprint(u.ID), "email": u.Email}) }
	return u, nil // Return created user (password omitted in JSON due to json:"-").
}

// Login validates credentials and issues a signed JWT.
func (s *userService) Login(req models.LoginRequest, jwtSecret string, exp time.Duration) (string, error) {
	// Look up by email; return invalid on any error (don't leak info).
	u, err := s.repo.FindByEmail(req.Email)
	if err != nil { // If not found or DB error, treat as invalid.
		if s.log != nil { s.log.Warn("login user not found", map[string]string{"email": req.Email}) }
		return "", errors.New("invalid credentials")
	}
	// Verify supplied password against stored bcrypt hash.
	if !utils.CheckPassword(u.Password, req.Password) {
		if s.log != nil { s.log.Warn("login wrong password", map[string]string{"email": req.Email}) }
		return "", errors.New("invalid credentials")
	}

	// Build JWT claims (subject, issued-at, expiration, plus optional email).
	claims := jwt.MapClaims{
		"sub": u.ID, // Subject: user ID.
		"exp": time.Now().Add(exp).Unix(), // Expiration time (unix seconds).
		"iat": time.Now().Unix(), // Issued-at (unix seconds).
		"eml": u.Email, // Optional claim to carry email.
	}
	// Create a token with HS256 signing method.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign the token with the shared secret.
	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil { // Log and propagate signing error.
		if s.log != nil { s.log.Error("login token sign error", map[string]string{"email": u.Email, "err": err.Error()}) }
		return "", err
	}

	// Log login success (helpful audit trail).
	if s.log != nil { s.log.Info("login success", map[string]string{"user_id": fmt.Sprint(u.ID), "email": u.Email}) }
	return signed, nil // Return compact JWT string.
}

// GetByID returns a user, preferring Redis cache and falling back to DB.
func (s *userService) GetByID(id uint) (*models.User, error) {
	// Try Redis first for speed.
	if s.rdb != nil { // Only if Redis configured.
		ctx := context.Background() // Context needed for Redis commands.
		key := s.cacheKeyUser(id) // Compose key like "user:1".
		if s.log != nil { s.log.Info("cache try GET", map[string]string{"key": key, "user_id": fmt.Sprint(id)}) }

		val, err := s.rdb.Get(ctx, key).Result() // Attempt GET.
		if err == nil { // Found a value (string).
			var u models.User // Destination struct.
			if json.Unmarshal([]byte(val), &u) == nil { // Decode JSON → struct.
				if s.log != nil { s.log.Info("cache HIT", map[string]string{"key": key, "user_id": fmt.Sprint(id)}) }
				return &u, nil // Return cached result immediately.
			}
			// If unmarshal failed, ignore cache and continue to DB.
			if s.log != nil { s.log.Warn("cache unmarshal failed", map[string]string{"key": key}) }
		} else if err == redis.Nil { // Key not present → MISS.
			if s.log != nil { s.log.Warn("cache MISS", map[string]string{"key": key, "user_id": fmt.Sprint(id)}) }
		} else { // Some other Redis error occurred.
			if s.log != nil { s.log.Error("cache GET error", map[string]string{"key": key, "err": err.Error()}) }
		}
	}

	// Fallback to DB if cache did not return a valid user.
	u, err := s.repo.FindByID(id) // Query DB.
	if err != nil { // Not found or DB error → propagate.
		if s.log != nil { s.log.Error("db fetch error in GetByID", map[string]string{"user_id": fmt.Sprint(id), "err": err.Error()}) }
		return nil, err
	}
	if s.log != nil { s.log.Info("db fetch success in GetByID", map[string]string{"user_id": fmt.Sprint(id)}) }

	// Store result in cache for next time.
	if s.rdb != nil { // Only if Redis configured.
		ctx := context.Background() // Redis context.
		key := s.cacheKeyUser(id) // Cache key again.
		if b, _ := json.Marshal(u); len(b) > 0 { // Marshal user to JSON.
			if err := s.rdb.Set(ctx, key, b, userCacheTTL).Err(); err == nil { // SET key value with TTL.
				if s.log != nil { s.log.Info("cache SET", map[string]string{"key": key, "user_id": fmt.Sprint(id), "ttl": userCacheTTL.String()}) }
			} else { // Log cache SET failure if it happens.
				if s.log != nil { s.log.Error("cache SET error", map[string]string{"key": key, "err": err.Error()}) }
			}
		}
	}
	return u, nil // Return the DB result.
}

// ---------------- CRUD ----------------

// CreateUser — admin-style create; use same semantics as Register.
func (s *userService) CreateUser(req models.RegisterRequest) (*models.User, error) {
	if s.log != nil { s.log.Info("CreateUser called", map[string]string{"email": req.Email}) } // Trace call.
	return s.Register(req) // Reuse register path for uniqueness & hashing logic.
}

// GetUser — explicit method name for CRUD; same as GetByID.
func (s *userService) GetUser(id uint) (*models.User, error) {
	if s.log != nil { s.log.Info("GetUser called", map[string]string{"user_id": fmt.Sprint(id)}) } // Trace call.
	return s.GetByID(id) // Reuse existing cache-aware read.
}

// UpdateUser applies partial updates; re-hashes password if provided; refreshes cache.
func (s *userService) UpdateUser(id uint, req models.UpdateUserRequest) (*models.User, error) {
	if s.log != nil { s.log.Info("UpdateUser called", map[string]string{"user_id": fmt.Sprint(id)}) } // Trace call.

	// Load current user state.
	u, err := s.repo.FindByID(id)
	if err != nil {
		if s.log != nil { s.log.Error("UpdateUser not found", map[string]string{"user_id": fmt.Sprint(id), "err": err.Error()}) }
		return nil, err
	}

	// Apply provided changes.
	if req.Name != nil { // Update name if provided.
		u.Name = core.NormalizeName(*req.Name) // Normalize new name.
	}
	if req.Email != nil { // If email change requested...
		if *req.Email != u.Email { // Only if it's different.
			if _, err := s.repo.FindByEmail(*req.Email); err == nil { // Check uniqueness.
				if s.log != nil { s.log.Warn("UpdateUser email exists", map[string]string{"email": *req.Email}) }
				return nil, errors.New("email already exists") // Abort on conflict.
			}
			u.Email = *req.Email // Apply new email.
		}
	}
	if req.Password != nil { // If new password provided...
		hash, err := utils.HashPassword(*req.Password) // Hash it.
		if err != nil {
			if s.log != nil { s.log.Error("UpdateUser hash error", map[string]string{"user_id": fmt.Sprint(id), "err": err.Error()}) }
			return nil, err
		}
		u.Password = hash // Store hashed password.
	}

	// Persist the update.
	if err := s.repo.Update(u); err != nil { // Write to DB.
		if s.log != nil { s.log.Error("UpdateUser db error", map[string]string{"user_id": fmt.Sprint(id), "err": err.Error()}) }
		return nil, err
	}

	// Refresh cache: delete the old value and set new.
	if s.rdb != nil {
		ctx := context.Background() // Redis context.
		key := s.cacheKeyUser(id) // Cache key.
		_ = s.rdb.Del(ctx, key).Err() // Best-effort invalidate; ignore error.
		if b, _ := json.Marshal(u); len(b) > 0 { // Marshal updated user.
			_ = s.rdb.Set(ctx, key, b, userCacheTTL).Err() // Best-effort set; ignore error.
		}
		if s.log != nil { s.log.Info("UpdateUser cache refreshed", map[string]string{"key": key}) } // Log cache refresh.
	}

	// Return updated user.
	return u, nil
}

// DeleteUser removes a user and deletes any cache entry.
func (s *userService) DeleteUser(id uint) error {
	if s.log != nil { s.log.Info("DeleteUser called", map[string]string{"user_id": fmt.Sprint(id)}) } // Trace call.

	// Delete from DB (returns ErrRecordNotFound if not present).
	if err := s.repo.Delete(id); err != nil {
		if s.log != nil { s.log.Error("DeleteUser db error", map[string]string{"user_id": fmt.Sprint(id), "err": err.Error()}) }
		return err
	}

	// Delete cache key to avoid stale reads.
	if s.rdb != nil {
		ctx := context.Background() // Redis context.
		_ = s.rdb.Del(ctx, s.cacheKeyUser(id)).Err() // Best-effort delete.
	}

	// Log success.
	if s.log != nil { s.log.Info("DeleteUser success", map[string]string{"user_id": fmt.Sprint(id)}) }
	return nil // Done.
}

// ListUsers returns a paginated page of users and total count.
func (s *userService) ListUsers(page, limit int) (*models.PagedUsers, error) {
	if s.log != nil { s.log.Info("ListUsers called", map[string]string{"page": fmt.Sprint(page), "limit": fmt.Sprint(limit)}) } // Trace.

	// Sanitize inputs: default page=1, limit=10..100
	if page < 1 { page = 1 } // Avoid zero/negative page.
	if limit <= 0 || limit > 100 { limit = 10 } // Clamp page size.

	// Compute offset for SQL LIMIT/OFFSET.
	offset := (page - 1) * limit // Skip previous pages.

	// Query repository for items + total.
	items, total, err := s.repo.List(offset, limit)
	if err != nil { // Propagate DB error to handler.
		if s.log != nil { s.log.Error("ListUsers db error", map[string]string{"err": err.Error()}) }
		return nil, err
	}

	// Compose response envelope with items & paging info.
	resp := &models.PagedUsers{Items: items, Total: total, Page: page, Limit: limit}

	// Optional log of result size (useful for monitoring).
	if s.log != nil { s.log.Info("ListUsers success", map[string]string{"count": fmt.Sprint(len(items)), "total": fmt.Sprint(total)}) }

	// Return page.
	return resp, nil
}
