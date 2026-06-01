package models

import (
	"strings"
	"time"
)

const StatusActive = "ACTIVE"
const StatusDeleted = "DELETED"

type CreateWaitingRoomRequest struct {
	MaxActiveUsersCount int    `json:"maxActiveUsersCount"`
	OriginApplication   string `json:"originApplication"`
}

type WaitingRoom struct {
	RoomId              string    `json:"roomId"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
	Status              string    `json:"status"`
	MaxActiveUsersCount int       `json:"maxActiveUsersCount"`
	OriginApplication   string    `json:"originApplication"`
}

type GetWaitingRoomRequest struct {
	RoomId string
}

type DeleteWaitingRoomRequest struct {
	RoomId       string
	IsSoftDelete bool
}

type CreateWaitingRoomValidationResult struct {
	ValidationError string
}

type ValidationError struct {
	Messages []string
}

type WaitingRoomStatus struct {
	RoomId               string `json:"roomId"`
	Decision             string `json:"decision"`
	Origin               string `json:"origin"`
	NumberOfActiveUsers  int64  `json:"numberOfActiveUsers"`
	NumberOfWaitingUsers int64  `json:"numberOfWaitingUsers"`
}

func (e *ValidationError) Error() string {
	return strings.Join(e.Messages, "\n")
}

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}
