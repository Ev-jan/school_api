package sqlconnect

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"schoolapi/internal/models"
	"schoolapi/pkg/utils"
	"strconv"

	"golang.org/x/crypto/argon2"
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
		if newExec.Password == "" {
			return nil, utils.ErrorHandler(errors.New("exec's password is blank"), "please log in")
		}
		//encrypt and store the provided password
		salt := make([]byte, 16)
		if _, err = rand.Read(salt); err != nil {
			return nil, utils.ErrorHandler(errors.New("failed to generate salt"), "error adding data")
		}
		hash := argon2.IDKey([]byte(newExec.Password), salt, 1, 64*1024, 4, 32)
		saltBase64 := base64.StdEncoding.EncodeToString(salt)
		hashBase64 := base64.StdEncoding.EncodeToString(hash)
		encodedHash := fmt.Sprintf("%s.%s", saltBase64, hashBase64)
		newExec.Password = encodedHash

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
	if err = db.QueryRow(query, req.Username).Scan(&user.FirstName, &user.LastName, &user.Email, &user.Username, &user.Password, &user.StatusInactive, &user.Role); err != nil {
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
