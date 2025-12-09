package operations

import (
	"testing"

	"github.com/aquamarinepk/aqm"
)

func TestDecodeSuccessResponse(t *testing.T) {
	tests := []struct {
		name    string
		resp    *aqm.SuccessResponse
		wantErr bool
	}{
		{
			name:    "nilResponse",
			resp:    nil,
			wantErr: true,
		},
		{
			name: "validMapResponse",
			resp: &aqm.SuccessResponse{
				Data: map[string]interface{}{
					"name": "Test",
					"id":   "123",
				},
			},
			wantErr: false,
		},
		{
			name: "validSliceResponse",
			resp: &aqm.SuccessResponse{
				Data: []interface{}{
					map[string]interface{}{"id": "1"},
					map[string]interface{}{"id": "2"},
				},
			},
			wantErr: false,
		},
		{
			name: "emptyDataResponse",
			resp: &aqm.SuccessResponse{
				Data: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dest interface{}
			err := decodeSuccessResponse(tt.resp, &dest)

			if (err != nil) != tt.wantErr {
				t.Errorf("decodeSuccessResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeSuccessResponseIntoStruct(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}

	resp := &aqm.SuccessResponse{
		Data: map[string]interface{}{
			"name": "Test Name",
			"id":   "test-id-123",
		},
	}

	var dest testStruct
	err := decodeSuccessResponse(resp, &dest)

	if err != nil {
		t.Fatalf("decodeSuccessResponse() error = %v", err)
	}

	if dest.Name != "Test Name" {
		t.Errorf("Name = %v, want %v", dest.Name, "Test Name")
	}
	if dest.ID != "test-id-123" {
		t.Errorf("ID = %v, want %v", dest.ID, "test-id-123")
	}
}

func TestDecodeSuccessResponseWithMeta(t *testing.T) {
	resp := &aqm.SuccessResponse{
		Data: map[string]string{"key": "value"},
		Meta: map[string]interface{}{"total": 100},
	}

	var dest map[string]string
	err := decodeSuccessResponse(resp, &dest)

	if err != nil {
		t.Fatalf("decodeSuccessResponse() error = %v", err)
	}

	if dest["key"] != "value" {
		t.Errorf("dest[key] = %v, want value", dest["key"])
	}
}
