package models

import (
	"strings"
	"time"
)

const StatusActive = "ACTIVE"
const StatusDeleted = "DELETED"
const DefaultActiveSessionTtlInSeconds = 10
const DefaultWaitingSessionTtlInSeconds = 600

type CreateWaitingRoomRequest struct {
	MaxActiveUsersCount      int    `json:"maxActiveUsersCount"`      // Required
	OriginApplication        string `json:"originApplication"`        // Required
	ActiveSessionTtlSeconds  int    `json:"activeSessionTtlSeconds"`  // Optional
	WaitingSessionTtlSeconds int    `json:"waitingSessionTtlSeconds"` // Optional
}

const BadRequestCode = "BadRequest"
const InternalServerErrorCode = "InternalServerError"

type ResponseError struct {
	Code    string                    `json:"code"`
	Message string                    `json:"message"`
	Details []ResponseErrorDetailItem `json:"details"`
}

type ResponseErrorDetailItem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrorItem struct {
	Field   string
	Message string
}

type WaitingRoom struct {
	RoomId                   string    `json:"roomId"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
	Status                   string    `json:"status"`
	MaxActiveUsersCount      int       `json:"maxActiveUsersCount"`
	OriginApplication        string    `json:"originApplication"`
	ActiveSessionTtlSeconds  int       `json:"activeSessionTtlSeconds"`
	WaitingSessionTtlSeconds int       `json:"waitingSessionTtlSeconds"`
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
	Messages []ValidationErrorItem
}

type WaitingRoomStatus struct {
	RoomId               string `json:"roomId"`
	Decision             string `json:"decision"`
	Origin               string `json:"origin"`
	NumberOfActiveUsers  int64  `json:"numberOfActiveUsers"`
	NumberOfWaitingUsers int64  `json:"numberOfWaitingUsers"`
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Messages))
	for _, message := range e.Messages {
		messages = append(messages, message.Message)
	}
	return strings.Join(messages, "\n")
}

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}
