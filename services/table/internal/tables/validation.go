package tables

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

func ValidateTableCreate(ctx context.Context, req TableCreateRequest) []string {
	var errors []string

	if strings.TrimSpace(req.Number) == "" {
		errors = append(errors, "number is required")
	}

	return errors
}

func ValidateTableUpdate(ctx context.Context, id uuid.UUID, req TableUpdateRequest) []string {
	var errors []string

	if id == uuid.Nil {
		errors = append(errors, "invalid table id")
	}

	if req.Status != "" {
		validStatuses := []string{"available", "open", "reserved", "cleaning", "out_of_service"}
		valid := false
		for _, s := range validStatuses {
			if req.Status == s {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, "invalid status")
		}
	}

	return errors
}

func ValidateOrderItemCreate(ctx context.Context, req OrderItemCreateRequest) []string {
	var errors []string

	if req.OrderID == uuid.Nil {
		errors = append(errors, "order_id is required")
	}

	if strings.TrimSpace(req.DishName) == "" {
		errors = append(errors, "dish_name is required")
	}

	if req.Quantity <= 0 {
		errors = append(errors, "quantity must be greater than 0")
	}

	if req.Price < 0 {
		errors = append(errors, "price cannot be negative")
	}

	return errors
}

func ValidateReservationCreate(ctx context.Context, req ReservationCreateRequest) []string {
	var errors []string

	if req.GuestCount <= 0 {
		errors = append(errors, "guest_count must be greater than 0")
	}

	if req.ReservedFor.IsZero() {
		errors = append(errors, "reserved_for is required")
	}

	if strings.TrimSpace(req.ContactName) == "" {
		errors = append(errors, "contact_name is required")
	}

	if strings.TrimSpace(req.ContactInfo) == "" {
		errors = append(errors, "contact_info is required")
	}

	return errors
}
