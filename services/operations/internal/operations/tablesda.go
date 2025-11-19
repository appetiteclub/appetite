package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/aquamarinepk/aqm"
)

// tableResource mirrors the payload returned by the table service.
type tableResource struct {
	ID          string             `json:"id"`
	Number      string             `json:"number"`
	Status      string             `json:"status"`
	GuestCount  int                `json:"guest_count"`
	AssignedTo  *string            `json:"assigned_to"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	CurrentBill *tableBillResource `json:"current_bill"`
}

type tableBillResource struct {
	Total float64 `json:"total"`
}

// orderGroupResource represents table-level billing groups used by the order UI.
type orderGroupResource struct {
	ID      string `json:"id"`
	TableID string `json:"table_id"`
	Name    string `json:"name"`
}

// TableDataAccess centralizes decoding of table service responses.
type TableDataAccess struct {
	client *aqm.ServiceClient
}

func NewTableDataAccess(client *aqm.ServiceClient) *TableDataAccess {
	return &TableDataAccess{client: client}
}

func (da *TableDataAccess) ListTables(ctx context.Context) ([]tableResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("table client not configured")
	}

	resp, err := da.client.List(ctx, "tables")
	if err != nil {
		return nil, err
	}

	var tables []tableResource
	if err := decodeSuccessResponse(resp, &tables); err != nil {
		return nil, err
	}

	return tables, nil
}

func (da *TableDataAccess) GetTable(ctx context.Context, id string) (*tableResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("table client not configured")
	}

	resp, err := da.client.Get(ctx, "tables", id)
	if err != nil {
		return nil, err
	}

	var table tableResource
	if err := decodeSuccessResponse(resp, &table); err != nil {
		return nil, err
	}

	return &table, nil
}

func (da *TableDataAccess) ListTableGroups(ctx context.Context, tableID string) ([]orderGroupResource, error) {
	if da == nil || da.client == nil {
		return nil, fmt.Errorf("table client not configured")
	}
	if tableID == "" {
		return nil, fmt.Errorf("missing table id")
	}

	path := fmt.Sprintf("/tables/%s/groups", tableID)
	resp, err := da.client.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var groups []orderGroupResource
	if err := decodeSuccessResponse(resp, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}
