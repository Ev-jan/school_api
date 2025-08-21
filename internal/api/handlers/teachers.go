package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"schoolapi/internal/models"
	"schoolapi/internal/repository/sqlconnect"
	"schoolapi/pkg/utils"
	"strconv"
	"time"
)

func GetTeachers(w http.ResponseWriter, r *http.Request) {
	var teachers []models.Teacher
	teachers, err := sqlconnect.GetTeachersDB(teachers, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Status string           `json:"status"`
		Count  int              `json:"count"`
		Data   []models.Teacher `json:"data"`
	}{"success", len(teachers), teachers}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func GetTeacher(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Error converting teacher ID", http.StatusInternalServerError)
		return
	}

	teacher, err := sqlconnect.GetTeacherDB(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(teacher); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func AddTeachers(w http.ResponseWriter, r *http.Request) {

	var newTeachers []models.Teacher
	if err := json.NewDecoder(r.Body).Decode(&newTeachers); err != nil {
		http.Error(w, "Invalid request body: ", http.StatusBadRequest)
		return
	}

	addedTeachers, err := sqlconnect.AddTeachersDB(newTeachers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var updatedTeacher models.Teacher

	if err := json.NewDecoder(r.Body).Decode(&updatedTeacher); err != nil {
		log.Printf("Error decoding json data: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updatedTeacher, err = sqlconnect.UpdateTeacherDB(ctx, id, updatedTeacher)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	existingTeacher, err := sqlconnect.PatchTeacherDB(id, updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(existingTeacher); err != nil {
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func PatchTeachers(w http.ResponseWriter, r *http.Request) {

	var updates []map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Printf("Error decoding json: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := sqlconnect.PatchTeachersDB(updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	err = sqlconnect.DeleteTeacherDB(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

	var ids []int
	err := json.NewDecoder(r.Body).Decode(&ids)
	if err != nil {
		http.Error(w, "Invalid teacher ids", http.StatusInternalServerError)
		return
	}
	deletedIds, err := sqlconnect.DeleteTeachersDB(ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status     string `json:"status"`
		DeletedIds []int  `json:"deleted_ids"`
	}{"Teachers successfully deleted", deletedIds}
	json.NewEncoder(w).Encode(response)
}

func GetStudentsByTeacherId(w http.ResponseWriter, r *http.Request) {
	teacherId := r.PathValue("id")

	var students []models.Student

	students, err := sqlconnect.GetStudentsByTeacherIdDB(teacherId, students)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Status string           `json:"status"`
		Count  int              `json:"count"`
		Data   []models.Student `json:"data"`
	}{
		"success", len(students), students,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding student count", http.StatusInternalServerError)
		return
	}
}

func GetStudentsCountByTeacherId(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Context())
	_, err := utils.AuthorizeUser(r.Context().Value(utils.ContextKey("role")).(string), "admin", "manager", "exec")
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	teacherId := r.PathValue("id")
	count, err := sqlconnect.GetStudentsCountByTeacherIdDB(teacherId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := struct {
		Status       string `json:"status"`
		TeacherID    string `json:"teacher_id"`
		StudentCount uint   `json:"student_count"`
	}{"success", teacherId, count}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding student count", http.StatusInternalServerError)
		return
	}
}
