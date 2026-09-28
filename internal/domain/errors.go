package domain

import "errors"

var (
	ErrTripNotFound        = errors.New("Trip not found")
	ErrTripAlreadyComplete = errors.New("Trip already complete")
	ErrDriverBusy          = errors.New("Driver already has an active trip")
	ErrIdempotencyConflict = errors.New("Conflict idempotency key")
)
