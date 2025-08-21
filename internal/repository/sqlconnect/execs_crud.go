package sqlconnect

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"schoolapi/internal/models"
	"schoolapi/pkg/utils"
	"strconv"
	"time"

	"github.com/go-mail/mail/v2"
)

func GetExecDB(id int) (models.Exec, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "error connecting to DB")
	}
	defer db.Close()
	var exec models.Exec
	err = db.QueryRow("SELECT id, first_name, last_name, email, username, user_created_at, status_inactive, role FROM execs WHERE id = ?", id).Scan(&exec.ID, &exec.FirstName, &exec.LastName, &exec.Email, &exec.Username, &exec.UserCreatedAt, &exec.StatusInactive, &exec.Role)
	if err == sql.ErrNoRows {
		return models.Exec{}, utils.ErrorHandler(err, "Exec not found")
	} else if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Database query error")
	}
	return exec, nil
}

func GetExecsDB(execs []models.Exec, r *http.Request) ([]models.Exec, error) {

	query := "SELECT id, first_name, last_name, email, username, user_created_at, status_inactive, role FROM execs WHERE 1=1"
	var args []any

	query, args = addFilters(r, query, args)
	query = addSorting(r, query)

	db, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "error connecting to DB")
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var exec models.Exec
		err := rows.Scan(&exec.ID, &exec.FirstName, &exec.LastName, &exec.Email, &exec.Username, &exec.UserCreatedAt, &exec.StatusInactive, &exec.Role)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			return nil, err
		}
		execs = append(execs, exec)
	}
	return execs, nil
}

func AddExecsDB(newExecs []models.Exec) ([]models.Exec, error) {
	db, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "error connecting to DB")
	}
	defer db.Close()

	stmt, err := db.Prepare(generateInsertQuery(models.Exec{}, "execs"))
	if err != nil {
		return nil, utils.ErrorHandler(err, "error preparing SQL statement")
	}
	defer stmt.Close()

	addedExecs := make([]models.Exec, len(newExecs))
	for i, newExec := range newExecs {
		// check if password exists
		newExec.Password, err = utils.HashPassword(newExec.Password)
		if err != nil {
			return nil, utils.ErrorHandler(err, "erro adding exec into db")
		}

		values := getStructValues(newExec)
		res, err := stmt.Exec(values...)
		if err != nil {
			return nil, utils.ErrorHandler(err, "error inserting data into DB")
		}
		lastId, err := res.LastInsertId()
		if err != nil {
			return nil, utils.ErrorHandler(err, "error getting last inserted ID")
		}
		newExec.ID = int(lastId)
		addedExecs[i] = newExec
	}
	return addedExecs, nil
}

func PatchExecDB(id int, updates map[string]any) (models.Exec, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	var existingExec models.Exec
	err = db.QueryRow("SELECT id, first_name, last_name, email, username, role FROM execs WHERE id = ?", id).Scan(&existingExec.ID, &existingExec.FirstName, &existingExec.LastName, &existingExec.Email, &existingExec.Username, &existingExec.Role)
	if err == sql.ErrNoRows {
		return models.Exec{}, utils.ErrorHandler(err, "Exec data not found")
	} else if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Failed to retrieve exec data")
	}

	execVal := reflect.ValueOf(&existingExec).Elem()
	execType := execVal.Type()

	for k, v := range updates {
		for i := 0; i < execVal.NumField(); i++ {
			field := execType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				execVal.Field(i).Set(reflect.ValueOf(v).Convert(execVal.Field(i).Type()))
			}
		}
	}

	if _, err = db.Exec("UPDATE execs SET first_name = ?, last_name = ?, email = ?, username = ?, role = ? WHERE id = ?", &existingExec.FirstName, &existingExec.LastName, &existingExec.Email, &existingExec.Username, &existingExec.Role, &existingExec.ID); err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "error updating exec")
	}
	return existingExec, nil
}

func PatchExecsDB(updates []map[string]any) error {
	db, err := ConnectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, update := range updates {
		idStr, ok := update["id"].(string)
		if !ok {
			tx.Rollback()
			return err
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			tx.Rollback()
			return err
		}

		var execFromDb models.Exec
		err = db.QueryRow("SELECT id, first_name, last_name, email, username, role FROM execs WHERE id = ?", id).Scan(&execFromDb.ID, &execFromDb.FirstName, &execFromDb.LastName, &execFromDb.Email, &execFromDb.Username, &execFromDb.Role)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return utils.ErrorHandler(err, "Exec not found in the database")
			}
			return utils.ErrorHandler(err, "error patching exec information")
		}

		execVal := reflect.ValueOf(&execFromDb).Elem()
		execType := execVal.Type()

		for k, v := range update {
			if k == "id" {
				continue // skip updating the ID field
			}
			for i := 0; i < execVal.NumField(); i++ {
				field := execType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := execVal.Field(i)
					if fieldVal.CanSet() {
						val := reflect.ValueOf(v)
						if val.Type().ConvertibleTo(fieldVal.Type()) {
							fieldVal.Set(val.Convert(fieldVal.Type()))
						} else {
							tx.Rollback()
							log.Printf("cannot convert %v to %v", val.Type(), fieldVal.Type())
							return utils.ErrorHandler(err, "Failed to patch exec information")
						}
					}
					break
				}
			}
		}

		_, err = tx.Exec("UPDATE execs SET first_name = ?, last_name = ?, email = ?, username = ?, role = ? WHERE id = ?", execFromDb.FirstName, execFromDb.LastName, execFromDb.Email, execFromDb.Username, execFromDb.Role, execFromDb.ID)
		if err != nil {
			tx.Rollback()
			return utils.ErrorHandler(err, "Failed to patch exec information")
		}
	}

	err = tx.Commit()
	if err != nil {
		return utils.ErrorHandler(err, "Failed to patch exec information")
	}
	return nil
}

func DeleteExecDB(id int) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM execs WHERE id = ?", id)
	if err != nil {
		return utils.ErrorHandler(err, "error deleting exec")
	}
	n, err := result.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "error deleting exec")
	}

	if n == 0 {
		return utils.ErrorHandler(err, "Exec not found")
	}
	return nil
}

func GetUserByUsername(w http.ResponseWriter, req models.Exec) (models.Exec, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "internal error")
	}
	defer db.Close()

	var user models.Exec
	query := `SELECT id, first_name, last_name, email, username, password, status_inactive, role FROM execs WHERE username = ?`
	if err = db.QueryRow(query, req.Username).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Username, &user.Password, &user.StatusInactive, &user.Role); err != nil {
		if err == sql.ErrNoRows {
			return models.Exec{}, utils.ErrorHandler(err, "user does not exist")
		}
		return models.Exec{}, utils.ErrorHandler(err, "error retrieving data")
	}

	if user.StatusInactive {
		return models.Exec{}, utils.ErrorHandler(err, "account is inactive")
	}
	return user, nil
}

func UpdatePasswordDB(id, currentPassword, updatedPassword string) (string, string, error) {

	db, err := ConnectDB()
	if err != nil {
		return "", "", utils.ErrorHandler(err, "internal error")
	}
	defer db.Close()

	var username, userpassword, userRole string
	err = db.QueryRow("SELECT username, password, role FROM execs WHERE id = ?", id).Scan(&username, &userpassword, &userRole)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", utils.ErrorHandler(err, "user not found")
		}
		return "", "", utils.ErrorHandler(err, "internal error")
	}

	err = utils.VerifyPassword(currentPassword, userpassword)
	if err != nil {
		return "", "", utils.ErrorHandler(err, "provided password does not match current password")
	}

	hashedPassword, err := utils.HashPassword(updatedPassword)
	if err != nil {
		return "", "", utils.ErrorHandler(err, "internal error")
	}

	currentTime := time.Now().Format(time.RFC3339)

	_, err = db.Exec("UPDATE execs SET password = ?, password_changed_at = ? WHERE id = ?", hashedPassword, currentTime, id)
	if err != nil {
		return "", "", utils.ErrorHandler(err, "error updating password")
	}

	return username, userRole, nil
}

func ForgotPasswordDB(email string) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "internal error")
	}
	defer db.Close()

	var exec models.Exec
	if err = db.QueryRow("SELECT id FROM execs WHERE email = ?", email).Scan(&exec.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.ErrorHandler(err, "user not found")
		}
		return utils.ErrorHandler(err, "internal error")
	}

	resetJtwDuration, err := strconv.Atoi(os.Getenv("RESET_JWT_EXP_DURATION"))
	if err != nil {
		return utils.ErrorHandler(err, "failed to sent password reset email")
	}

	mins := time.Duration(resetJtwDuration)
	expiry := time.Now().Add(mins * time.Minute).Format(time.RFC3339)
	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		return utils.ErrorHandler(err, "failed to sent password reset email")
	}
	// token is sent to the user (as part of link), whilst hasnedTokenString is stored in the DB.
	// when user hits the reset url, we extract that token, generate a hash string from it, and check if the resulting hash matches the one we stored in the db.
	// this prevents malicious users from resetting someone's passwords

	token := hex.EncodeToString(tokenBytes)
	hashedToken := sha256.Sum256(tokenBytes)
	hashedTokenString := hex.EncodeToString(hashedToken[:])

	if _, err = db.Exec("UPDATE execs SET password_reset_token = ?, password_token_expires = ? WHERE id = ?", hashedTokenString, expiry, exec.ID); err != nil {
		return utils.ErrorHandler(err, "failed to send password reset email")
	}

	// Send the reset email
	host := os.Getenv("HOST_ADDRESS")
	port := os.Getenv("API_PORT")
	resetLinkDuration := os.Getenv("RESET_JWT_EXP_DURATION")
	resetUrl := fmt.Sprintf("https://%s%s/execs/reset-password/reset/%s", host, port, token)
	message := fmt.Sprintf("Forgot your password? Reset your password using the following link: \n %s \nIf you did request a password reset, please ignore this email. This link is only valid for %s minutes", resetUrl, resetLinkDuration)

	m := mail.NewMessage()
	m.SetHeader("From", "schooladmin@school.com")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Your password reset link")
	m.SetBody("text/plain", message)

	dialer := mail.NewDialer(host, 1025, "", "")
	if err = dialer.DialAndSend(m); err != nil {
		return utils.ErrorHandler(err, "failed to sent password reset email")
	}
	return nil
}

func ResetPasswordDB(token, newPassword string) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "internal error")
	}
	defer db.Close()

	var user models.Exec
	bytes, err := hex.DecodeString(token)
	if err != nil {
		return utils.ErrorHandler(err, "internal error")
	}

	hashedToken := sha256.Sum256(bytes)
	hashedTokenString := hex.EncodeToString(hashedToken[:])

	query := `SELECT id, email FROM execs WHERE password_reset_token = ? AND password_token_expires > ?`
	if err := db.QueryRow(query, hashedTokenString, time.Now().Format(time.RFC3339)).Scan(&user.ID, &user.Email); err != nil {
		return utils.ErrorHandler(err, "invalid or expired reset token")
	}

	// hash the new password

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return utils.ErrorHandler(err, "Internal error")
	}
	passwordChangeDate := time.Now().Format(time.RFC3339)

	if _, err := db.Exec("UPDATE execs SET password = ?, password_reset_token = NULL, password_token_expires = NULL, password_changed_at = ? WHERE id = ?", hashedPassword, passwordChangeDate, user.ID); err != nil {
		return utils.ErrorHandler(err, "failed to change password")
	}
	return nil
}
