package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"schoolapi/internal/models"
	"schoolapi/internal/repository/sqlconnect"
	"strconv"
	"strings"
)

func GetTeachers(w http.ResponseWriter, r *http.Request) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		http.Error(w, "Error connecting to DB: ", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	query := "SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE 1=1"
	var args []any

	query, args = addFilters(r, query, args)
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

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Query error: %v", err)
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	teacherList := make([]models.Teacher, 0)
	for rows.Next() {
		var teacher models.Teacher
		err := rows.Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		teacherList = append(teacherList, teacher)
	}

	response := struct {
		Status string           `json:"status"`
		Count  int              `json:"count"`
		Data   []models.Teacher `json:"data"`
	}{"success", len(teacherList), teacherList}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func GetTeacher(w http.ResponseWriter, r *http.Request) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		http.Error(w, "Error connecting to DB: ", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Error converting teacher ID", http.StatusInternalServerError)
		return
	}

	var teacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
	if err == sql.ErrNoRows {
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(teacher); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func AddTeachers(w http.ResponseWriter, r *http.Request) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		http.Error(w, "Error connecting to DB: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var newTeachers []models.Teacher
	if err := json.NewDecoder(r.Body).Decode(&newTeachers); err != nil {
		http.Error(w, "Invalid request body: ", http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO teachers (first_name, last_name, email, `class`, `subject`) VALUES (?,?,?,?,?)")
	if err != nil {
		http.Error(w, "Error preparing SQL statement: ", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	addedTeachers := make([]models.Teacher, len(newTeachers))
	for i, t := range newTeachers {
		res, err := stmt.Exec(t.FirstName, t.LastName, t.Email, t.Class, t.Subject)
		if err != nil {
			http.Error(w, "Error inserting data into DB: ", http.StatusInternalServerError)
			return
		}
		lastId, err := res.LastInsertId()
		if err != nil {
			http.Error(w, "Error getting last inserted ID: ", http.StatusInternalServerError)
			return
		}
		t.ID = int(lastId)
		addedTeachers[i] = t
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		Status string           `json:"status"`
		Count  int              `json:"count"`
		Data   []models.Teacher `json:"data"`
	}{
		Status: "success",
		Count:  len(addedTeachers),
		Data:   addedTeachers,
	})
}

func UpdateTeacher(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting teacher's id string to int: %v", err)
		http.Error(w, "Invalid teacher id", http.StatusBadRequest)
		return
	}

	var updatedTeacher models.Teacher

	if err := json.NewDecoder(r.Body).Decode(&updatedTeacher); err != nil {
		log.Printf("Error decoding json data: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var existingTeacher models.Teacher
	err = db.QueryRowContext(r.Context(), "SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		http.Error(w, "Teacher data not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error retrieving teacher %d: %v", id, err)
		http.Error(w, "Failed to retrieve teacher data", http.StatusInternalServerError)
		return
	}

	updatedTeacher.ID = existingTeacher.ID
	if _, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", &updatedTeacher.FirstName, &updatedTeacher.LastName, &updatedTeacher.Email, &updatedTeacher.Class, &updatedTeacher.Subject, &updatedTeacher.ID); err != nil {
		log.Printf("Error updating teacher: %v", err)
		http.Error(w, "Error updating teacher", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedTeacher); err != nil {
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func PatchTeacher(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting teacher's id string to int: %v", err)
		http.Error(w, "Invalid teacher id", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Printf("Error decoding json data: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var existingTeacher models.Teacher
	err = db.QueryRowContext(r.Context(), "SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		http.Error(w, "Teacher data not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error retrieving teacher %d: %v", id, err)
		http.Error(w, "Failed to retrieve teacher data", http.StatusInternalServerError)
		return
	}

	// Apply updates

	// for k, v := range updates {
	// 	switch k {
	// 	case "first_name":
	// 		existingTeacher.FirstName = v.(string)
	// 	case "last_name":
	// 		existingTeacher.LastName = v.(string)
	// 	case "email":
	// 		existingTeacher.Email = v.(string)
	// 	case "class":
	// 		existingTeacher.Class = v.(string)
	// 	case "subject":
	// 		existingTeacher.Subject = v.(string)
	// 	}
	// }

	// Apply updates using reflect

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
		http.Error(w, "Error updating teacher", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(existingTeacher); err != nil {
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func PatchTeachers(w http.ResponseWriter, r *http.Request) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var updates []map[string]any
	if err = json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Printf("Error decoding json: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error connecting DB %v", err)
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return
	}

	for _, update := range updates {
		fmt.Println("Update:", update)
		idRaw, ok := update["id"].(float64)
		if !ok {
			tx.Rollback()
			log.Println("Error reading id in update")
			http.Error(w, "Invalid teacher id", http.StatusBadRequest)
			return
		}

		id := int(idRaw)
		fmt.Println("ID", id)
		if err != nil {
			tx.Rollback()
			log.Printf("Error converting string id into int %v", err)
			http.Error(w, "Invalid teacher id", http.StatusBadRequest)
			return
		}

		var teacherFromDB models.Teacher
		if err = db.QueryRowContext(r.Context(), "SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(
			&teacherFromDB.ID,
			&teacherFromDB.FirstName,
			&teacherFromDB.LastName,
			&teacherFromDB.Email,
			&teacherFromDB.Class,
			&teacherFromDB.Subject,
		); err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				http.Error(w, "Teacher not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Erro retrieving teacher", http.StatusInternalServerError)
			return
		}

		// Apply updates using Reflect
		teacherVal := reflect.ValueOf(&teacherFromDB).Elem()
		teacherType := teacherVal.Type()

		for k, v := range update {
			if k == "id" {
				continue // skip updating the id field
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
							log.Printf("Cannot convert %v to %v", val.Type(), fieldVal.Type())
							return
						}
					}
					break
				}
			}
		}

		if _, err := tx.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?",
			teacherFromDB.FirstName,
			teacherFromDB.LastName,
			teacherFromDB.Email,
			teacherFromDB.Class,
			teacherFromDB.Subject,
			teacherFromDB.ID,
		); err != nil {
			http.Error(w, "Error updating teacher", http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Error updating teachers", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func DeleteTeacher(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting teacher's id string to int: %v", err)
		http.Error(w, "Invalid teacher id", http.StatusBadRequest)
		return
	}

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	result, err := db.ExecContext(r.Context(), "DELETE FROM teachers WHERE id = ?", id)
	if err != nil {
		log.Printf("Error deleting teacher: %v", err)
		http.Error(w, "Error deleting teacher", http.StatusInternalServerError)
		return
	}
	n, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error retrieving rows affected: %v", err)
		http.Error(w, "Error deleting teacher", http.StatusInternalServerError)
		return
	}

	if n == 0 {
		http.Error(w, "Teacher not found", http.StatusNotFound)
	}

	// w.WriteHeader(http.StatusNoContent)
	response := struct {
		Status string `json:"status"`
		ID     int    `json:"id"`
	}{"Teacher successfully deleted", id}

	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response data: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

}

func DeleteTeachers(w http.ResponseWriter, r *http.Request) {

	db, err := sqlconnect.ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		http.Error(w, "DB connection failed", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var ids []int
	err = json.NewDecoder(r.Body).Decode(&ids)
	if err != nil {
		log.Printf("Error reading ids from request body: %v", err)
		http.Error(w, "Invalid teacher ids", http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Failed to start db transaction", http.StatusInternalServerError)
		return
	}

	stmt, err := tx.Prepare("DELETE FROM teachers WHERE id = ?")
	if err != nil {
		tx.Rollback()
		log.Printf("Error preparing query to delete teachers: %v", err)
		http.Error(w, "Failed to prepare delete statement", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	deletedIds := []int{}
	for _, id := range ids {
		result, err := stmt.Exec(id)
		if err != nil {
			tx.Rollback()
			log.Printf("Error deleting teacher: %v", err)
			http.Error(w, "Failed to delete teacher", http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			tx.Rollback()
			log.Printf("Error deleting teacher: %v", err)
			http.Error(w, "Failed to delete teacher", http.StatusInternalServerError)
			return
		}

		// If teacher was deleted, then add the id to the deletedIds
		if rowsAffected > 0 {
			deletedIds = append(deletedIds, id)
		}

		if rowsAffected < 1 {
			tx.Rollback()
			http.Error(w, fmt.Sprintf("ID %d does not exist", id), http.StatusBadRequest)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Transaction failed, could not delete teachers: %v", err)
		http.Error(w, "Transaction failed, could not delete teachers", http.StatusInternalServerError)
		return
	}

	if len(deletedIds) < 1 {
		http.Error(w, "Teacher IDs do not exist", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status     string `json:"status"`
		DeletedIds []int  `json:"deleted_ids"`
	}{
		"Teachers successfully deleted", deletedIds,
	}
	json.NewEncoder(w).Encode(response)
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
