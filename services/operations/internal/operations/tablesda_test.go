package operations

import (
	"context"
	"testing"
)

func TestNewTableDataAccess(t *testing.T) {
	da := NewTableDataAccess(nil)
	if da == nil {
		t.Error("NewTableDataAccess() returned nil")
	}
}

func TestTableDataAccessListTablesNilClient(t *testing.T) {
	da := &TableDataAccess{client: nil}

	_, err := da.ListTables(context.Background())
	if err == nil {
		t.Error("ListTables() with nil client should return error")
	}
}

func TestTableDataAccessListTablesNilDA(t *testing.T) {
	var da *TableDataAccess

	_, err := da.ListTables(context.Background())
	if err == nil {
		t.Error("ListTables() with nil DA should return error")
	}
}

func TestTableDataAccessGetTableNilClient(t *testing.T) {
	da := &TableDataAccess{client: nil}

	_, err := da.GetTable(context.Background(), "table-1")
	if err == nil {
		t.Error("GetTable() with nil client should return error")
	}
}

func TestTableDataAccessGetTableNilDA(t *testing.T) {
	var da *TableDataAccess

	_, err := da.GetTable(context.Background(), "table-1")
	if err == nil {
		t.Error("GetTable() with nil DA should return error")
	}
}
