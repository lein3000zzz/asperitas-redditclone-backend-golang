package user

import (
	"database/sql"
	"errors"
	"redditclone/pkg/utils"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

const (
	testUser = "testuser"
	password = "password123"
)

func TestUserMySQLRepo_Authorize_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	defer utils.CloseDB(db)

	repo := NewMySQLRepo(db)

	username := testUser
	password := password
	expectedID := "user1"

	rows := sqlmock.NewRows([]string{"id", "username", "password"}).
		AddRow(expectedID, username, password)
	mock.ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").
		WithArgs(username).
		WillReturnRows(rows)

	user, err := repo.Authorize(username, password)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedID, user.ID)
	assert.Equal(t, username, user.Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQLRepo_Authorize_UserNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer utils.CloseDB(db)

	repo := NewMySQLRepo(db)

	username := "unknownuser"
	password := password

	mock.ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").
		WithArgs(username).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.Authorize(username, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.EqualError(t, err, ErrNoUser.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQLRepo_Authorize_BadPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	defer utils.CloseDB(db)

	repo := NewMySQLRepo(db)

	username := testUser
	correctPasswordInDB := "correct_password"
	incorrectPasswordInput := "incorrect_password"
	expectedID := "user1"

	rows := sqlmock.NewRows([]string{"id", "username", "password"}).
		AddRow(expectedID, username, correctPasswordInDB)
	mock.ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").
		WithArgs(username).
		WillReturnRows(rows)

	user, err := repo.Authorize(username, incorrectPasswordInput)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.EqualError(t, err, ErrBadPass.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQLRepo_Authorize_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer utils.CloseDB(db)

	repo := NewMySQLRepo(db)

	username := testUser
	password := password
	dbError := errors.New("database connection error")

	mock.ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").
		WithArgs(username).
		WillReturnError(dbError)

	user, err := repo.Authorize(username, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.EqualError(t, err, ErrNoUser.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQLRepo_Register_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer utils.CloseDB(db)

	repo := NewMySQLRepo(db)

	username := "newuser"
	password := "newpassword"

	countRows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users WHERE username = \\?").
		WithArgs(username).
		WillReturnRows(countRows)

	// \\ - нужно для регекса

	mock.ExpectExec("INSERT INTO users \\(id, username, password\\) VALUES \\(\\?, \\?, \\?\\)").
		WithArgs(sqlmock.AnyArg(), username, password).
		WillReturnResult(sqlmock.NewResult(1, 1))

	user, err := repo.Register(username, password)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, password, user.Password)
	assert.NotEmpty(t, user.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQLRepo_Register_AlreadyExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer utils.CloseDB(db)

	repo := NewMySQLRepo(db)

	username := "existinguser"
	password := "password"

	countRows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users WHERE username = \\?").
		WithArgs(username).
		WillReturnRows(countRows)

	user, err := repo.Register(username, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.EqualError(t, err, ErrAlreadyExists.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQLRepo_Register_CheckExistsDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer utils.CloseDB(db)

	repo := NewMySQLRepo(db)

	username := testUser
	password := password
	dbError := errors.New("DB error on count")

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users WHERE username = \\?").
		WithArgs(username).
		WillReturnError(dbError)

	user, err := repo.Register(username, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.EqualError(t, err, dbError.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMySQLRepo_Register_InsertDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer utils.CloseDB(db)

	repo := NewMySQLRepo(db)

	username := "newuser"
	password := "newpassword"
	dbError := errors.New("DB error on insert")

	countRows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users WHERE username = \\?").
		WithArgs(username).
		WillReturnRows(countRows)

	mock.ExpectExec("INSERT INTO users \\(id, username, password\\) VALUES \\(\\?, \\?, \\?\\)").
		WithArgs(sqlmock.AnyArg(), username, password).
		WillReturnError(dbError)

	user, err := repo.Register(username, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.EqualError(t, err, dbError.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}
