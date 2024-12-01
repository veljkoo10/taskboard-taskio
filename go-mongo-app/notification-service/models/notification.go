package models

import (
	"encoding/json"
	"fmt"
	"github.com/gocql/gocql"
	"io"
	"time"
)

type NotificationStatus string

const (
	Unread NotificationStatus = "unread"
	Read   NotificationStatus = "read"
)

type Notification struct {
	ID        gocql.UUID         `json:"id"`
	UserID    string             `json:"user_id"`
	Message   string             `json:"message"`
	CreatedAt time.Time          `json:"created_at"`
	IsActive  bool               `json:"is_active"`
	Status    NotificationStatus `json:"status"`
}

type Notifications []*Notification

func (o *Notification) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(o)
}

func (o *Notification) FromJSON(r io.Reader) error {
	d := json.NewDecoder(r)
	return d.Decode(o)
}

func (ns NotificationStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(ns))
}

func (ns *NotificationStatus) UnmarshalJSON(data []byte) error {
	var status string
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	switch status {
	case "unread", "read":
		*ns = NotificationStatus(status)
		return nil
	}

	return fmt.Errorf("invalid notification status: %s", status)
}

func (n *Notification) Validate() error {
	if n.Status != Unread && n.Status != Read {
		return fmt.Errorf("invalid status: %s", n.Status)
	}
	if n.UserID == "" || n.Message == "" {
		return fmt.Errorf("user_id and message cannot be empty")
	}
	return nil
}
