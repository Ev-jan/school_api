package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"schoolapi/internal/models"
	"schoolapi/internal/repository/sqlconnect"
	"strconv"
)

func GetStudents(w http.ResponseWriter, r *http.Request) {
	limit, page := getPaginationParams(r)

	var students []models.Student
	students, totalCount, err := sqlconnect.GetStudentsDB(students, r, limit, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Status     string           `json:"status"`
		TotalCount int              `json:"total_count"`
		Page       int              `json:"page"`
		Limit      int              `json:"limit"`
		Data       []models.Student `json:"data"`
	}{"success", totalCount, page, limit, students}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func GetStudent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Error converting student ID", http.StatusInternalServerError)
		return
	}

	student, err := sqlconnect.GetStudentDB(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(student); err != nil {
		log.Printf("JSON encoding error: %v", err)
	}
}

func AddStudents(w http.ResponseWriter, r *http.Request) {

	var newStudents []models.Student
	if err := json.NewDecoder(r.Body).Decode(&newStudents); err != nil {
		http.Error(w, "Invalid request body: ", http.StatusBadRequest)
		return
	}

	addedStudents, err := sqlconnect.AddStudentsDB(newStudents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		Status string           `json:"status"`
		Count  int              `json:"count"`
		Data   []models.Student `json:"data"`
	}{
		Status: "success",
		Count:  len(addedStudents),
		Data:   addedStudents,
	})
}

func UpdateStudent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting student's id string to int: %v", err)
		http.Error(w, "Invalid student id", http.StatusBadRequest)
		return
	}

	var updatedStudent models.Student

	if err := json.NewDecoder(r.Body).Decode(&updatedStudent); err != nil {
		log.Printf("Error decoding json data: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	updatedStudent, err = sqlconnect.UpdateStudentDB(id, updatedStudent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedStudent); err != nil {
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func PatchStudent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting student's id string to int: %v", err)
		http.Error(w, "Invalid student id", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Printf("Error decoding json data: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	existingStudent, err := sqlconnect.PatchStudentDB(id, updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(existingStudent); err != nil {
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func PatchStudents(w http.ResponseWriter, r *http.Request) {

	var updates []map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Printf("Error decoding json: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := sqlconnect.PatchStudentsDB(updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func DeleteStudent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting student's id string to int: %v", err)
		http.Error(w, "Invalid student id", http.StatusBadRequest)
		return
	}

	err = sqlconnect.DeleteStudentDB(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := struct {
		Status string `json:"status"`
		ID     int    `json:"id"`
	}{"Student successfully deleted", id}

	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response data: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

}

func DeleteStudents(w http.ResponseWriter, r *http.Request) {

	var ids []int
	err := json.NewDecoder(r.Body).Decode(&ids)
	if err != nil {
		log.Printf("Error reading ids from request body: %v", err)
		http.Error(w, "Invalid student ids", http.StatusInternalServerError)
		return
	}

	deletedIds, err := sqlconnect.DeleteStudentsDB(ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := struct {
		Status     string `json:"status"`
		DeletedIds []int  `json:"deleted_ids"`
	}{"Students successfully deleted", deletedIds}
	json.NewEncoder(w).Encode(response)
}
