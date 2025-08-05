package sqlconnect

import (
	"database/sql"
	"log"
	"net/http"
	"reflect"
	"schoolapi/internal/models"
	"strconv"
	"strings"
)

func GetTeacherDB(id int) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		// http.Error(w, "Error connecting to DB: ", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	defer db.Close()
	var teacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
	if err == sql.ErrNoRows {
		// http.Error(w, "Teacher not found", http.StatusNotFound)
		return models.Teacher{}, err
	} else if err != nil {
		// http.Error(w, "Database query error", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	return teacher, nil
}

func GetTeachersDB(teachers []models.Teacher, r *http.Request) ([]models.Teacher, error) {

	query := "SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE 1=1"
	var args []any

	query, args = addFilters(r, query, args)
	query = addSorting(r, query)

	db, err := ConnectDB()
	if err != nil {
		// http.Error(w, "Error connecting to DB: ", http.StatusInternalServerError)
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	// teacherList := make([]models.Teacher, 0)
	for rows.Next() {
		var teacher models.Teacher
		err := rows.Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			return nil, err
		}
		teachers = append(teachers, teacher)
	}
	return teachers, nil
}

func AddTeachersDB(newTeachers []models.Teacher) ([]models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		// http.Error(w, "Error connecting to DB: "+err.Error(), http.StatusInternalServerError)
		return nil, err
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO teachers (first_name, last_name, email, `class`, `subject`) VALUES (?,?,?,?,?)")
	if err != nil {
		// http.Error(w, "Error preparing SQL statement: ", http.StatusInternalServerError)
		return nil, err
	}
	defer stmt.Close()

	addedTeachers := make([]models.Teacher, len(newTeachers))
	for i, t := range newTeachers {
		res, err := stmt.Exec(t.FirstName, t.LastName, t.Email, t.Class, t.Subject)
		if err != nil {
			// http.Error(w, "Error inserting data into DB: ", http.StatusInternalServerError)
			return nil, err
		}
		lastId, err := res.LastInsertId()
		if err != nil {
			// http.Error(w, "Error getting last inserted ID: ", http.StatusInternalServerError)
			return nil, err
		}
		t.ID = int(lastId)
		addedTeachers[i] = t
	}
	return addedTeachers, nil
}

func UpdateTeacherDB(id int, updatedTeacher models.Teacher) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		// http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	defer db.Close()

	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		// http.Error(w, "Teacher data not found", http.StatusNotFound)
		return models.Teacher{}, err
	} else if err != nil {
		log.Printf("Error retrieving teacher %d: %v", id, err)
		// http.Error(w, "Failed to retrieve teacher data", http.StatusInternalServerError)
		return models.Teacher{}, err
	}

	updatedTeacher.ID = existingTeacher.ID
	if _, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", &updatedTeacher.FirstName, &updatedTeacher.LastName, &updatedTeacher.Email, &updatedTeacher.Class, &updatedTeacher.Subject, &updatedTeacher.ID); err != nil {
		log.Printf("Error updating teacher: %v", err)
		// http.Error(w, "Error updating teacher", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	return updatedTeacher, nil
}

func PatchTeacherDB(id int, updates map[string]any) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		// http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	defer db.Close()

	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		// http.Error(w, "Teacher data not found", http.StatusNotFound)
		return models.Teacher{}, err
	} else if err != nil {
		log.Printf("Error retrieving teacher %d: %v", id, err)
		// http.Error(w, "Failed to retrieve teacher data", http.StatusInternalServerError)
		return models.Teacher{}, err
	}

	teacherVal := reflect.ValueOf(&existingTeacher).Elem()
	teacherType := teacherVal.Type()

	for k, v := range updates {
		for i := 0; i < teacherVal.NumField(); i++ {
			field := teacherType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				teacherVal.Field(i).Set(reflect.ValueOf(v).Convert(teacherVal.Field(i).Type()))
			}
		}
	}

	if _, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject, &existingTeacher.ID); err != nil {
		log.Printf("Error updating teacher: %v", err)
		// http.Error(w, "Error updating teacher", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	return existingTeacher, nil
}

func PatchTeachersDB(updates []map[string]any) error {
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

		var teacherFromDb models.Teacher
		err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacherFromDb.ID, &teacherFromDb.FirstName, &teacherFromDb.LastName, &teacherFromDb.Email, &teacherFromDb.Class, &teacherFromDb.Subject)
		if err != nil {
			log.Println("ID:", id)
			log.Printf("Type: %T", id)
			log.Println(err)
			tx.Rollback()
			if err == sql.ErrNoRows {
				return err
			}
			return err
		}

		teacherVal := reflect.ValueOf(&teacherFromDb).Elem()
		teacherType := teacherVal.Type()

		for k, v := range update {
			if k == "id" {
				continue // skip updating the ID field
			}
			for i := 0; i < teacherVal.NumField(); i++ {
				field := teacherType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := teacherVal.Field(i)
					if fieldVal.CanSet() {
						val := reflect.ValueOf(v)
						if val.Type().ConvertibleTo(fieldVal.Type()) {
							fieldVal.Set(val.Convert(fieldVal.Type()))
						} else {
							tx.Rollback()
							log.Printf("cannot convert %v to %v", val.Type(), fieldVal.Type())
							return err
						}
					}
					break
				}
			}
		}

		_, err = tx.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", teacherFromDb.FirstName, teacherFromDb.LastName, teacherFromDb.Email, teacherFromDb.Class, teacherFromDb.Subject, teacherFromDb.ID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func DeleteTeacherDB(id int) error {
	db, err := ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		// http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return err
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM teachers WHERE id = ?", id)
	if err != nil {
		log.Printf("Error deleting teacher: %v", err)
		// http.Error(w, "Error deleting teacher", http.StatusInternalServerError)
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error retrieving rows affected: %v", err)
		// http.Error(w, "Error deleting teacher", http.StatusInternalServerError)
		return err
	}

	if n == 0 {
		// http.Error(w, "Teacher not found", http.StatusNotFound)
		return err
	}
	return nil
}

// Helpers

func addSorting(r *http.Request, query string) string {
	sortParams := r.URL.Query()["sort_by"] // NB! this approach for getting a slice of strings instead of one big string
	if len(sortParams) > 0 {
		var orderClauses []string
		for _, param := range sortParams {
			parts := strings.Split(param, ":")
			if len(parts) != 2 {
				continue
			}
			field, order := parts[0], parts[1]
			if !isValidSortField(field) || !isValidSortOrder(order) {
				continue
			}
			orderClauses = append(orderClauses, field+" "+order)
		}
		if len(orderClauses) > 0 {
			query += " ORDER BY " + strings.Join(orderClauses, ", ")
		}
	}
	return query
}

func isValidSortOrder(order string) bool {
	return order == "asc" || order == "desc"
}

func isValidSortField(field string) bool {
	validFields := map[string]bool{
		"first_name": true,
		"last_name":  true,
		"email":      true,
		"class":      true,
		"subject":    true,
	}
	return validFields[field]
}

func addFilters(r *http.Request, query string, args []any) (string, []any) {
	params := map[string]string{
		"first_name": "first_name",
		"last_name":  "last_name",
		"email":      "email",
		"class":      "class",
		"subject":    "subject",
	}

	for param, dbField := range params {
		value := r.URL.Query().Get(param)
		if value != "" {
			query += " AND " + dbField + " = ?"
			args = append(args, value)
		}
	}
	return query, args
}

func DeleteTeachersDB(ids []int) ([]int, error) {
	db, err := ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		// http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return nil, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		// http.Error(w, "Failed to start db transaction", http.StatusInternalServerError)
		return nil, err
	}

	stmt, err := tx.Prepare("DELETE FROM teachers WHERE id = ?")
	if err != nil {
		tx.Rollback()
		log.Printf("Error preparing query to delete teachers: %v", err)
		// http.Error(w, "Failed to prepare delete statement", http.StatusInternalServerError)
		return nil, err
	}
	defer stmt.Close()

	deletedIds := []int{}
	for _, id := range ids {
		result, err := stmt.Exec(id)
		if err != nil {
			tx.Rollback()
			log.Printf("Error deleting teacher: %v", err)
			// http.Error(w, "Failed to delete teacher", http.StatusInternalServerError)
			return nil, err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			tx.Rollback()
			log.Printf("Error deleting teacher: %v", err)
			// http.Error(w, "Failed to delete teacher", http.StatusInternalServerError)
			return nil, err
		}

		// If teacher was deleted, then add the id to the deletedIds
		if rowsAffected > 0 {
			deletedIds = append(deletedIds, id)
		}

		if rowsAffected < 1 {
			tx.Rollback()
			// http.Error(w, fmt.Sprintf("ID %d does not exist", id), http.StatusBadRequest)
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Transaction failed, could not delete teachers: %v", err)
		// http.Error(w, "Transaction failed, could not delete teachers", http.StatusInternalServerError)
		return nil, err
	}

	if len(deletedIds) < 1 {
		// http.Error(w, "Teacher IDs do not exist", http.StatusBadRequest)
		return nil, err
	}
	return deletedIds, nil
}
