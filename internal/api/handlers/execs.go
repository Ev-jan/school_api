package handlers

import (
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

func GetExecs(w http.ResponseWriter, r *http.Request) {
	var execs []models.Exec
	execs, err := sqlconnect.GetExecsDB(execs, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Status string        `json:"status"`
		Count  int           `json:"count"`
		Data   []models.Exec `json:"data"`
	}{"success", len(execs), execs}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func GetExec(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Error converting exec ID", http.StatusInternalServerError)
		return
	}

	exec, err := sqlconnect.GetExecDB(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(exec); err != nil {
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func AddExecs(w http.ResponseWriter, r *http.Request) {

	var newExecs []models.Exec
	if err := json.NewDecoder(r.Body).Decode(&newExecs); err != nil {
		http.Error(w, "Invalid request body: ", http.StatusBadRequest)
		return
	}

	addedExecs, err := sqlconnect.AddExecsDB(newExecs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		Status string        `json:"status"`
		Count  int           `json:"count"`
		Data   []models.Exec `json:"data"`
	}{
		Status: "success",
		Count:  len(addedExecs),
		Data:   addedExecs,
	})
}

func PatchExec(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting exec's id string to int: %v", err)
		http.Error(w, "Invalid exec id", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Printf("Error decoding json data: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	existingExec, err := sqlconnect.PatchExecDB(id, updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(existingExec); err != nil {
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func PatchExecs(w http.ResponseWriter, r *http.Request) {

	var updates []map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		log.Printf("Error decoding json: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := sqlconnect.PatchExecsDB(updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func DeleteExec(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("Error converting exec's id string to int: %v", err)
		http.Error(w, "Invalid exec id", http.StatusBadRequest)
		return
	}

	err = sqlconnect.DeleteExecDB(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := struct {
		Status string `json:"status"`
		ID     int    `json:"id"`
	}{"Exec successfully deleted", id}

	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response data: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

}

func Login(w http.ResponseWriter, r *http.Request) {
	var req models.Exec

	//validate request data
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// validate parsed credentials

	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	user, err := sqlconnect.GetUserByUsername(w, req)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusInternalServerError)
		return
	}

	if err := utils.VerifyPassword(req.Password, user.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := utils.SignToken(strconv.Itoa(user.ID), req.Username, user.Role)
	if err != nil {
		http.Error(w, "failed to create authorization token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(time.Hour * 24),
		SameSite: http.SameSiteStrictMode,
	})

	response := struct {
		Token string `json:"token"`
	}{tokenString}

	json.NewEncoder(w).Encode(response)

}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Unix(0, 0),
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "logged out successfully"}`))
}

func UpdatePassword(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var req models.UpdatePasswordRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, "Please enter password", http.StatusBadRequest)
		return
	}

	username, userRole, err := sqlconnect.UpdatePasswordDB(idStr, req.CurrentPassword, req.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := utils.SignToken(idStr, username, userRole)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Bearer",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(time.Hour * 24),
		SameSite: http.SameSiteStrictMode,
	})

	response := struct {
		Message string `json:"message"`
	}{"Password has been succesfully updated"}

	json.NewEncoder(w).Encode(response)
}

func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	if req.Email == "" {
		http.Error(w, "please enter email address", http.StatusBadRequest)
		return
	}

	if err := sqlconnect.ForgotPasswordDB(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// respond with success message
	fmt.Fprintf(w, "Password reset link sent to %s", req.Email)
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("resetcode")

	type request struct {
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	if req.NewPassword == "" || req.ConfirmPassword == "" {
		http.Error(w, "please enter new password and confirm password", http.StatusBadRequest)
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		http.Error(w, "password shoud match", http.StatusBadRequest)
		return
	}

	if err := sqlconnect.ResetPasswordDB(token, req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintln(w, "Password successfully reset")
}
