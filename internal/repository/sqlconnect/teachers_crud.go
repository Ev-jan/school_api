package sqlconnect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"schoolapi/internal/models"
	"schoolapi/pkg/utils"
	"strconv"
	"strings"
)

func GetTeacherDB(id int) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error connecting to DB")
	}
	defer db.Close()
	var teacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, subject FROM teachers WHERE id = ?", id).Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Subject)
	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Teacher not found")
	} else if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Database query error")
	}

	var classes []models.Class
	query := `SELECT c.id, c.class_name FROM classes c INNER JOIN class_assignments ca ON c.id = ca.class_id WHERE ca.teacher_id = ?`

	rows, err := db.Query(query, id)
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error retrieving teacher details")
	}
	defer rows.Close()

	for rows.Next() {
		var class models.Class
		if err := rows.Scan(&class.ID, &class.ClassName); err != nil {
			return models.Teacher{}, utils.ErrorHandler(err, "Error scanning a class row")
		}
		classes = append(classes, class)
	}
	teacher.Classes = classes

	return teacher, nil
}

func GetTeachersDB(teachers []models.Teacher, r *http.Request) ([]models.Teacher, error) {

	query := "SELECT id, first_name, last_name, email, subject FROM teachers WHERE 1=1"
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
		err := rows.Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Subject)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			return nil, err
		}

		var classes []models.Class
		query = `SELECT c.id, c.class_name FROM classes c INNER JOIN class_assignments ca ON c.id = ca.class_id WHERE ca.teacher_id = ?`

		rows, err = db.Query(query, teacher.ID)
		if err != nil {
			utils.ErrorHandler(err, "Error retrieving teacher details")
		}
		defer rows.Close()

		for rows.Next() {
			var class models.Class
			if err := rows.Scan(&class.ID, &class.ClassName); err != nil {
				utils.ErrorHandler(err, "Error scanning a class row")
			}
			classes = append(classes, class)
		}
		teacher.Classes = classes
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

func UpdateTeacherDB(ctx context.Context, id int, updatedTeacher models.Teacher) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		return models.Teacher{}, utils.ErrorHandler(err, "DB connection failed")
	}
	defer db.Close()

	// Ensure the teacher exists (and lock row minimally)
	var exists int
	if err := db.QueryRowContext(ctx, "SELECT 1 FROM teachers WHERE id = ?", id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Teacher{}, utils.ErrorHandler(err, "Teacher not found")
		}
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to verify teacher")
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to begin transaction")
	}
	defer func() {
		// safety: rollback if not committed
		_ = tx.Rollback()
	}()

	// 1) Update base fields
	if _, err := tx.ExecContext(ctx,
		`UPDATE teachers SET first_name=?, last_name=?, email=?, subject=? WHERE id=?`,
		updatedTeacher.FirstName, updatedTeacher.LastName, updatedTeacher.Email, updatedTeacher.Subject, id,
	); err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating teacher")
	}

	// 2) Gather desired class IDs from payload
	desired := make(map[int]struct{}, len(updatedTeacher.Classes))
	desiredIDs := make([]int, 0, len(updatedTeacher.Classes))
	for _, c := range updatedTeacher.Classes {
		if c.ID == 0 {
			return models.Teacher{}, utils.ErrorHandler(fmt.Errorf("missing class ID"), "Each class must have a valid ID")
		}
		if _, seen := desired[c.ID]; !seen {
			desired[c.ID] = struct{}{}
			desiredIDs = append(desiredIDs, c.ID)
		}
	}

	if len(desiredIDs) > 0 {
		placeholders := strings.Repeat("?,", len(desiredIDs))
		placeholders = placeholders[:len(placeholders)-1]
		args := make([]any, 0, len(desiredIDs))
		for _, id := range desiredIDs {
			args = append(args, id)
		}
		rows, err := tx.QueryContext(ctx, "SELECT id FROM classes WHERE id IN ("+placeholders+")", args...)
		if err != nil {
			return models.Teacher{}, utils.ErrorHandler(err, "Failed to validate class IDs")
		}
		valid := map[int]struct{}{}
		for rows.Next() {
			var cid int
			if scanErr := rows.Scan(&cid); scanErr != nil {
				rows.Close()
				return models.Teacher{}, utils.ErrorHandler(scanErr, "Failed to validate class IDs")
			}
			valid[cid] = struct{}{}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return models.Teacher{}, utils.ErrorHandler(err, "Failed to validate class IDs")
		}
		// Check mismatch
		for cid := range desired {
			if _, ok := valid[cid]; !ok {
				return models.Teacher{}, utils.ErrorHandler(fmt.Errorf("invalid class id: %d", cid), "One or more classes do not exist")
			}
		}
	}

	// 4) Load current assignments
	current := map[int]struct{}{}
	curRows, err := tx.QueryContext(ctx, `SELECT class_id FROM class_assignments WHERE teacher_id = ?`, id)
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to fetch current class assignments")
	}
	for curRows.Next() {
		var cid int
		if scanErr := curRows.Scan(&cid); scanErr != nil {
			curRows.Close()
			return models.Teacher{}, utils.ErrorHandler(scanErr, "Failed to fetch current class assignments")
		}
		current[cid] = struct{}{}
	}
	curRows.Close()
	if err := curRows.Err(); err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to fetch current class assignments")
	}

	// 5) Diff
	toInsert := make([]int, 0)
	for cid := range desired {
		if _, ok := current[cid]; !ok {
			toInsert = append(toInsert, cid)
		}
	}
	toDelete := make([]int, 0)
	for cid := range current {
		if _, ok := desired[cid]; !ok {
			toDelete = append(toDelete, cid)
		}
	}

	// 6) Apply deletes
	if len(toDelete) > 0 {
		placeholders := strings.Repeat("?,", len(toDelete))
		placeholders = placeholders[:len(placeholders)-1]
		args := make([]any, 0, len(toDelete)+1)
		args = append(args, id)
		for _, cid := range toDelete {
			args = append(args, cid)
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM class_assignments WHERE teacher_id=? AND class_id IN (`+placeholders+`)`,
			args...,
		); err != nil {
			return models.Teacher{}, utils.ErrorHandler(err, "Failed to remove class assignments")
		}
	}

	// 7) Apply inserts (bulk)
	if len(toInsert) > 0 {
		// Build VALUES (?, ?), (?, ?)...
		sb := strings.Builder{}
		sb.WriteString("INSERT INTO class_assignments (teacher_id, class_id) VALUES ")
		args := make([]any, 0, len(toInsert)*2)
		for i, cid := range toInsert {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("(?, ?)")
			args = append(args, id, cid)
		}
		// If there is a UNIQUE(teacher_id, class_id) you don't need ON DUPLICATE,
		// but it's safe to add to avoid race duplicates:
		// sb.WriteString(" ON DUPLICATE KEY UPDATE class_id=VALUES(class_id)")
		if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
			return models.Teacher{}, utils.ErrorHandler(err, "Failed to add class assignments")
		}
	}

	// 8) Commit
	if err := tx.Commit(); err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Transaction commit failed")
	}

	// 9) (Optional) Re-read canonical result to return
	updatedTeacher.ID = id
	// Reload classes to return a fresh slice
	rows, err := db.QueryContext(ctx, `
        SELECT c.id, c.class_name
        FROM classes c
        JOIN class_assignments ca ON ca.class_id = c.id
        WHERE ca.teacher_id = ?
        ORDER BY c.id
    `, id)
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to load updated classes")
	}
	defer rows.Close()
	updatedTeacher.Classes = nil
	for rows.Next() {
		var c models.Class
		if err := rows.Scan(&c.ID, &c.ClassName); err != nil {
			return models.Teacher{}, utils.ErrorHandler(err, "Failed to load updated classes")
		}
		updatedTeacher.Classes = append(updatedTeacher.Classes, c)
	}
	if err := rows.Err(); err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Failed to load updated classes")
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
	err = db.QueryRow("SELECT id, first_name, last_name, email, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Subject)
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

	if _, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, subject = ? WHERE id = ?", &existingTeacher.FirstName, &existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Subject, &existingTeacher.ID); err != nil {
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
		err = db.QueryRow("SELECT id, first_name, last_name, email, subject FROM teachers WHERE id = ?", id).Scan(&teacherFromDb.ID, &teacherFromDb.FirstName, &teacherFromDb.LastName, &teacherFromDb.Email, &teacherFromDb.Subject)
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

		_, err = tx.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, subject = ? WHERE id = ?", teacherFromDb.FirstName, teacherFromDb.LastName, teacherFromDb.Email, teacherFromDb.Subject, teacherFromDb.ID)
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

func GetStudentsByTeacherIdDB(teacherId string, students []models.Student) ([]models.Student, error) {
	db, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Failed connect to DB")
	}
	defer db.Close()

	query := `
	SELECT DISTINCT s.id, s.first_name, s.last_name, s.email
						FROM students s
						JOIN class_enrollments  ce ON ce.student_id = s.id
						JOIN class_assignments ca ON ca.class_id   = ce.class_id
						WHERE ca.teacher_id = ?
						ORDER BY s.id;`

	rows, err := db.Query(query, teacherId)
	if err != nil {
		return nil, utils.ErrorHandler(err, "Failed to retrieve teacher data from DB")
	}
	defer rows.Close()

	for rows.Next() {
		var student models.Student
		err := rows.Scan(&student.FirstName, &student.LastName, &student.Email, &student.Class)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Failed to retrieve teacher data from DB")
		}
		students = append(students, student)
	}

	err = rows.Err()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Failed to retrieve teacher data from DB")
	}
	return students, nil
}

func GetStudentsCountByTeacherIdDB(teacherId string) (uint, error) {
	db, err := ConnectDB()
	if err != nil {
		return 0, utils.ErrorHandler(err, "Failed connect to DB")
	}
	defer db.Close()

	var studentCount uint

	query := `SELECT COUNT(DISTINCT s.id) AS student_count
				FROM students s
				JOIN class_enrollments ce ON ce.student_id = s.id
				JOIN class_assignments ca ON ca.class_id = ce.class_id
				WHERE ca.teacher_id = ?`

	err = db.QueryRow(query, teacherId).Scan(&studentCount)
	if err != nil {
		return 0, utils.ErrorHandler(err, "Failed to retrieve student count")
	}
	return studentCount, nil
}
