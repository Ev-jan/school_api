package sqlconnect

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"schoolapi/internal/models"
	"schoolapi/pkg/utils"
	"strconv"
)

func GetStudentDB(id int) (models.Student, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error connecting to DB")
	}
	defer db.Close()
	var student models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Class)
	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Student not found")
	} else if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Database query error")
	}
	return student, nil
}

func GetStudentsDB(students []models.Student, r *http.Request, limit, page int) ([]models.Student, int, error) {
	query := "SELECT id, first_name, last_name, email, class FROM students WHERE 1=1"
	var args []any

	query, args = addFilters(r, query, args)
	query = addSorting(r, query)

	offset := (page - 1) * limit
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	db, err := ConnectDB()
	if err != nil {
		return nil, 0, utils.ErrorHandler(err, "Error connecting to DB")
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, utils.ErrorHandler(err, "internal error")
	}
	defer rows.Close()

	for rows.Next() {
		var student models.Student
		err := rows.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Class)
		if err != nil {
			return nil, 0, utils.ErrorHandler(err, "internal error")
		}
		students = append(students, student)
	}
	var totalCount int
	countQuery := "SELECT COUNT(DISTINCT id) FROM students"
	if err = db.QueryRow(countQuery).Scan(&totalCount); err != nil {
		return nil, 0, utils.ErrorHandler(err, "internal error")
	}

	return students, totalCount, nil
}

func AddStudentsDB(newStudents []models.Student) ([]models.Student, error) {
	db, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error connecting to DB")
	}
	defer db.Close()

	stmt, err := db.Prepare(generateInsertQuery(models.Student{}, "students"))
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error preparing SQL statement")
	}
	defer stmt.Close()

	addedStudents := make([]models.Student, len(newStudents))
	for i, t := range newStudents {
		values := getStructValues(t)
		res, err := stmt.Exec(values...)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error inserting data into DB")
		}
		lastId, err := res.LastInsertId()
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error getting last inserted ID")
		}
		t.ID = int(lastId)
		addedStudents[i] = t
	}
	return addedStudents, nil
}

func UpdateStudentDB(id int, updatedStudent models.Student) (models.Student, error) {
	db, err := ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		return models.Student{}, utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	var existingStudent models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&existingStudent.ID, &existingStudent.FirstName, &existingStudent.LastName, &existingStudent.Email, &existingStudent.Class)
	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Student data not found")
	} else if err != nil {
		log.Printf("Error retrieving student %d: %v", id, err)
		return models.Student{}, utils.ErrorHandler(err, "Failed to retrieve student data")
	}

	updatedStudent.ID = existingStudent.ID
	if _, err = db.Exec("UPDATE students SET first_name = ?, last_name = ?, email = ?, class = ? WHERE id = ?", &updatedStudent.FirstName, &updatedStudent.LastName, &updatedStudent.Email, &updatedStudent.Class, &updatedStudent.ID); err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating student")
	}
	return updatedStudent, nil
}

func PatchStudentDB(id int, updates map[string]any) (models.Student, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	var existingStudent models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&existingStudent.ID, &existingStudent.FirstName, &existingStudent.LastName, &existingStudent.Email, &existingStudent.Class)
	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Student data not found")
	} else if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Failed to retrieve student data")
	}

	studentVal := reflect.ValueOf(&existingStudent).Elem()
	studentType := studentVal.Type()

	for k, v := range updates {
		for i := 0; i < studentVal.NumField(); i++ {
			field := studentType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				studentVal.Field(i).Set(reflect.ValueOf(v).Convert(studentVal.Field(i).Type()))
			}
		}
	}

	if _, err = db.Exec("UPDATE students SET first_name = ?, last_name = ?, email = ?, class = ? WHERE id = ?", &existingStudent.FirstName, &existingStudent.LastName, &existingStudent.Email, &existingStudent.Class, &existingStudent.ID); err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating student")
	}
	return existingStudent, nil
}

func PatchStudentsDB(updates []map[string]any) error {
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

		var studentFromDb models.Student
		err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&studentFromDb.ID, &studentFromDb.FirstName, &studentFromDb.LastName, &studentFromDb.Email, &studentFromDb.Class)
		if err != nil {
			log.Println("ID:", id)
			log.Printf("Type: %T", id)
			log.Println(err)
			tx.Rollback()
			if err == sql.ErrNoRows {
				return utils.ErrorHandler(err, "Student not found in the database")
			}
			return utils.ErrorHandler(err, "Error patching student information")
		}

		studentVal := reflect.ValueOf(&studentFromDb).Elem()
		studentType := studentVal.Type()

		for k, v := range update {
			if k == "id" {
				continue // skip updating the ID field
			}
			for i := 0; i < studentVal.NumField(); i++ {
				field := studentType.Field(i)
				if field.Tag.Get("json") == k+",omitempty" {
					fieldVal := studentVal.Field(i)
					if fieldVal.CanSet() {
						val := reflect.ValueOf(v)
						if val.Type().ConvertibleTo(fieldVal.Type()) {
							fieldVal.Set(val.Convert(fieldVal.Type()))
						} else {
							tx.Rollback()
							log.Printf("cannot convert %v to %v", val.Type(), fieldVal.Type())
							return utils.ErrorHandler(err, "Failed to patch student information")
						}
					}
					break
				}
			}
		}

		_, err = tx.Exec("UPDATE students SET first_name = ?, last_name = ?, email = ?, class = ? WHERE id = ?", studentFromDb.FirstName, studentFromDb.LastName, studentFromDb.Email, studentFromDb.Class, studentFromDb.ID)
		if err != nil {
			tx.Rollback()
			return utils.ErrorHandler(err, "Failed to patch student information")
		}
	}

	err = tx.Commit()
	if err != nil {
		return utils.ErrorHandler(err, "Failed to patch student information")
	}
	return nil
}

func DeleteStudentDB(id int) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM students WHERE id = ?", id)
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting student")
	}
	n, err := result.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting student")
	}

	if n == 0 {
		return utils.ErrorHandler(err, "Student not found")
	}
	return nil
}

func DeleteStudentsDB(ids []int) ([]int, error) {
	db, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Failed to delete students")
	}

	stmt, err := tx.Prepare("DELETE FROM students WHERE id = ?")
	if err != nil {
		tx.Rollback()
		return nil, utils.ErrorHandler(err, "Failed to delete students")
	}
	defer stmt.Close()

	deletedIds := []int{}
	for _, id := range ids {
		result, err := stmt.Exec(id)
		if err != nil {
			tx.Rollback()
			return nil, utils.ErrorHandler(err, "Failed to delete students")
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			tx.Rollback()
			return nil, utils.ErrorHandler(err, "Failed to delete students")
		}

		// If student was deleted, add the id to the deletedIds
		if rowsAffected > 0 {
			deletedIds = append(deletedIds, id)
		}

		if rowsAffected < 1 {
			tx.Rollback()
			return nil, utils.ErrorHandler(err, fmt.Sprintf("ID %d does not exist", id))
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Transaction failed, could not delete students")
	}

	if len(deletedIds) < 1 {
		return nil, utils.ErrorHandler(err, "Student IDs do not exist")
	}
	return deletedIds, nil
}
