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

func GetTeacherDB(id int) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error connecting to DB")
	}
	defer db.Close()
	var teacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Teacher not found")
	} else if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Database query error")
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
		return nil, utils.ErrorHandler(err, "Error connecting to DB")
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil, err
	}
	defer rows.Close()

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
		return nil, utils.ErrorHandler(err, "Error connecting to DB")
	}
	defer db.Close()

	// stmt, err := db.Prepare("INSERT INTO teachers (first_name, last_name, email, `class`, `subject`) VALUES (?,?,?,?,?)") // the olden way of manual labor
	stmt, err := db.Prepare(generateInsertQuery(models.Teacher{}, "teachers")) // using new function
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error preparing SQL statement")
	}
	defer stmt.Close()

	addedTeachers := make([]models.Teacher, len(newTeachers))
	for i, t := range newTeachers {
		// res, err := stmt.Exec(t.FirstName, t.LastName, t.Email, t.Class, t.Subject)
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
		addedTeachers[i] = t
	}
	return addedTeachers, nil
}

func UpdateTeacherDB(id int, updatedTeacher models.Teacher) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		return models.Teacher{}, utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Teacher data not found")
	} else if err != nil {
		log.Printf("Error retrieving teacher %d: %v", id, err)
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to retrieve teacher data")
	}

	updatedTeacher.ID = existingTeacher.ID
	if _, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", &updatedTeacher.FirstName, &updatedTeacher.LastName, &updatedTeacher.Email, &updatedTeacher.Class, &updatedTeacher.Subject, &updatedTeacher.ID); err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating teacher")
	}
	return updatedTeacher, nil
}

func PatchTeacherDB(id int, updates map[string]any) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)
	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Teacher data not found")
	} else if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to retrieve teacher data")
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
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating teacher")
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
				return utils.ErrorHandler(err, "Teacher not found in the database")
			}
			return utils.ErrorHandler(err, "Error patching teacher information")
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
							return utils.ErrorHandler(err, "Failed to patch teacher information")
						}
					}
					break
				}
			}
		}

		_, err = tx.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", teacherFromDb.FirstName, teacherFromDb.LastName, teacherFromDb.Email, teacherFromDb.Class, teacherFromDb.Subject, teacherFromDb.ID)
		if err != nil {
			tx.Rollback()
			return utils.ErrorHandler(err, "Failed to patch teacher information")
		}
	}

	err = tx.Commit()
	if err != nil {
		return utils.ErrorHandler(err, "Failed to patch teacher information")
	}
	return nil
}

func DeleteTeacherDB(id int) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	result, err := db.Exec("DELETE FROM teachers WHERE id = ?", id)
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting teacher")
	}
	n, err := result.RowsAffected()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting teacher")
	}

	if n == 0 {
		return utils.ErrorHandler(err, "Teacher not found")
	}
	return nil
}

func DeleteTeachersDB(ids []int) ([]int, error) {
	db, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Failed to delete teachers")
	}

	stmt, err := tx.Prepare("DELETE FROM teachers WHERE id = ?")
	if err != nil {
		tx.Rollback()
		return nil, utils.ErrorHandler(err, "Failed to delete teachers")
	}
	defer stmt.Close()

	deletedIds := []int{}
	for _, id := range ids {
		result, err := stmt.Exec(id)
		if err != nil {
			tx.Rollback()
			return nil, utils.ErrorHandler(err, "Failed to delete teachers")
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			tx.Rollback()
			return nil, utils.ErrorHandler(err, "Failed to delete teachers")
		}

		// If teacher was deleted, then add the id to the deletedIds
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
		return nil, utils.ErrorHandler(err, "Transaction failed, could not delete teachers")
	}

	if len(deletedIds) < 1 {
		return nil, utils.ErrorHandler(err, "Teacher IDs do not exist")
	}
	return deletedIds, nil
}
