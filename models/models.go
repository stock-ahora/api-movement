package models

import (
	"time"

	"github.com/google/uuid"
)

type MovementsEvent struct {
	Id                 uuid.UUID            `json:"id"`
	ProductPerMovement []ProductPerMovement `json:"products"`
	RequestId          uuid.UUID            `json:"request_id"`
}

type ProductPerMovement struct {
	Id             string    `json:"id"`
	ProductID      uuid.UUID `json:"product_id"`
	Count          int       `json:"count"`
	MovementId     uuid.UUID `json:"movement_id"`
	DateLimit      time.Time `json:"date_limit"`
	MovementTypeId int       `json:"movement_type"`
	CreatedAt      time.Time `json:"created_at"`
}
