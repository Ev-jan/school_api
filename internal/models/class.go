package models

type Class struct {
	ID        int    `json:"id" db:"id"`
	ClassName string `json:"class_name" db:"class_name"`
}
