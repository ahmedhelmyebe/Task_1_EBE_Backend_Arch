// repository hides GORM details behind an interface—DB-agnostic.
// Data-access layer. Only talks to the database (via GORM here)-> (only talks to DB, no HTTP/JSON).
// Implements all repository operations the service needs: Create, FindByID/Email, Update, Delete, and List with total count. Clean and DB-agnostic.
// 3
package repositories

import (
	"HelmyTask/models" // Import our User model to map results.
	"errors"

	"gorm.io/gorm" // GORM DB type is injected so repos are testable/mocked.
)

// UserRepository defines the operations our service layer expects.
// Depending on interfaces (not concrete types) helps testability and swapping implementations.
type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
	//ADDIGN  THE reamin CRUD
	Update(user *models.User) error
	Delete(id uint) error                                 // Delete by primary key.
	List(offset, limit int) ([]models.User, int64, error) // Page through users + total count.

}

// privvv
// userRepo is a private struct implementing UserRepository.
// It holds a *gorm.DB that can connect to any dialect (mysql/postgres/sqlite/sqlserver).
type userRepo struct{ db *gorm.DB }

// NewUserRepository is a constructor that injects *gorm.DB and returns an interface.
// This allows main.go to wire dependencies without exposing concrete types to other layers.

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{db: db} // Simple constructor; easy to swap in tests.
}

// Create inserts a new user row using GORM's Create method.
func (r *userRepo) Create(u *models.User) error {
	return r.db.Create(u).Error // .Error exposes any DB error to caller.
}

// FindByEmail queries for a user with the given email.
// We use a parameterized query (WHERE email = ?) which GORM compiles safely for the dialect.
func (r *userRepo) FindByEmail(email string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil // Return pointer to the found user.
}

func (r *userRepo) FindByID(id uint) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, id).Error; err != nil { // First(&u, id) loads where primary key = id.
		return nil, err
	}
	return &u, nil
}

// Update saves fields on an existing user (assumes u has valid ID).
func (r *userRepo) Update(u *models.User) error {
	return r.db.Save(u).Error // Save writes all fields; for partial updates use Select/Omit.
}

// Delete removes a user row by primary key. If not found, return ErrRecordNotFound.
func (r *userRepo) Delete(id uint) error {
	res := r.db.Delete(&models.User{}, id) // Soft delete if GORM soft-deletes are enabled; here it's hard delete.
	if res.Error != nil {
		return res.Error                   // Return DB error if any.
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound      // No row to delete → treat as not found.
	}
	return nil
}

// List returns a page of users and the total count (for pagination UIs).
func (r *userRepo) List(offset, limit int) ([]models.User, int64, error) {
	var (
		items []models.User // Slice to collect this page.
		total int64         // Total rows in table.
	)
	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err // Counting failed → return error.
	}
	if err := r.db.
		Limit(limit).      // Restrict page size.
		Offset(offset).    // Start from offset (page-1)*limit.
		Order("id ASC").   // Deterministic ordering.
		Find(&items).      // Load rows into slice.
		Error; err != nil {
		return nil, 0, err // Find failed → return error.
	}
	return items, total, nil // Return slice and total count.
}

// Helper: IsNotFound checks GORM's "record not found" sentinel.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound) // True if wrapped or direct ErrRecordNotFound.
}
