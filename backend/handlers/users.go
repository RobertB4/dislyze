package handlers

import (
	"encoding/json"
	"net/http"
)

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UsersHandler struct{}

func NewUsersHandler() *UsersHandler {
	return &UsersHandler{}
}

func (h *UsersHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Hardcoded list of users for now
	users := []User{
		{
			ID:        "1",
			Email:     "john@example.com",
			FirstName: "John",
			LastName:  "Doe",
		},
		{
			ID:        "2",
			Email:     "jane@example.com",
			FirstName: "Jane",
			LastName:  "Smith",
		},
		{
			ID:        "3",
			Email:     "bob@example.com",
			FirstName: "Bob",
			LastName:  "Johnson",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
