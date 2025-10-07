package repositories

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"HelmyTask/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// helper: new GORM DB using a sqlmock connection with MySQL dialect.
func newMySQLMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	// Important: pass existing *sql.DB to gorm's mysql driver
	dial := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true, // we don't need to ping real server
	})

	gdb, err := gorm.Open(dial, &gorm.Config{})
	require.NoError(t, err)
	return gdb, mock, sqlDB
}

func TestUserRepository_Create(t *testing.T) {
	db, mock, sqlDB := newMySQLMockDB(t)
	defer sqlDB.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	// GORM INSERT: we match the table and columns. Exact SQL can differ slightly,
	// so we use a regexp with only the important bits.
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `users` (`name`,`email`,`password`,`created_at`,`updated_at`) VALUES (?,?,?,?,?)")).
		WithArgs("Ahmed", "a@b.c", "hash", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1)) // last insert id=1, affected=1
	mock.ExpectCommit()

	u := &models.User{Name: "Ahmed", Email: "a@b.c", Password: "hash", CreatedAt: now, UpdatedAt: now}
	err := repo.Create(u)
	require.NoError(t, err)
	assert.Equal(t, uint(1), u.ID) // GORM maps last insert id
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db, mock, sqlDB := newMySQLMockDB(t)
	defer sqlDB.Close()

	repo := NewUserRepository(db)

	email := "a@b.c"

	// Expect SELECT ... WHERE email = ?
	rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "created_at", "updated_at"}).
		AddRow(2, "Ahmed", "a@b.c", "hash", time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT * FROM `users` WHERE email = ? ORDER BY `users`.`id` LIMIT ?",
	)).WithArgs(email, sqlmock.AnyArg()).
		WillReturnRows(rows)

	u, err := repo.FindByEmail("a@b.c")
	require.NoError(t, err)
	assert.Equal(t, uint(2), u.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	db, mock, sqlDB := newMySQLMockDB(t)
	defer sqlDB.Close()
	repo := NewUserRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `users` WHERE `users`.`id` = ?")).
		WithArgs(999).
		WillReturnResult(sqlmock.NewResult(0, 0)) // RowsAffected = 0 -> not found
	mock.ExpectCommit()

	err := repo.Delete(999)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
