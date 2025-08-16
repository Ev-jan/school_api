package models

type Teacher struct {
	ID        int     `json:"id,omitempty" db:"id,omitempty"`
	FirstName string  `json:"first_name,omitempty" db:"first_name,omitempty"`
	LastName  string  `json:"last_name,omitempty" db:"last_name,omitempty"`
	Email     string  `json:"email,omitempty" db:"email,omitempty"`
	Classes   []Class `json:"classes,omitempty"`
	Subject   string  `json:"subject,omitempty" db:"subject,omitempty"`
}
