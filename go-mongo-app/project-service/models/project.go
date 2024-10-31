package models

type Project struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Owner       string `json:"owner"`
}
