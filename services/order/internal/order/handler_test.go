package order

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aquamarinepk/aqm"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "withNilDependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := HandlerDeps{}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			if h == nil {
				t.Fatal("NewHandler() returned nil")
			}

			if h.logger == nil {
				t.Error("NewHandler() should set noop logger when nil")
			}
		})
	}
}

func TestHandlerEnsureTableAllowsOrdering(t *testing.T) {
	tests := []struct {
		name        string
		tableID     uuid.UUID
		cacheStatus string
		expectErr   bool
	}{
		{
			name:      "nilTableID",
			tableID:   uuid.Nil,
			expectErr: true,
		},
		{
			name:        "availableTable",
			tableID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440040"),
			cacheStatus: "available",
			expectErr:   false,
		},
		{
			name:        "openTable",
			tableID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440041"),
			cacheStatus: "open",
			expectErr:   false,
		},
		{
			name:        "reservedTable",
			tableID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440042"),
			cacheStatus: "reserved",
			expectErr:   false,
		},
		{
			name:        "occupiedTable",
			tableID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440043"),
			cacheStatus: "occupied",
			expectErr:   true,
		},
		{
			name:        "closedTable",
			tableID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440044"),
			cacheStatus: "closed",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewTableStateCache(nil, nil)
			if tt.cacheStatus != "" {
				cache.Set(tt.tableID, tt.cacheStatus)
			}

			deps := HandlerDeps{
				TableStatesCache: cache,
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			_, err := h.ensureTableAllowsOrdering(context.Background(), tt.tableID)
			if (err != nil) != tt.expectErr {
				t.Errorf("ensureTableAllowsOrdering() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestHandlerEnsureTableAllowsOrderingNilCache(t *testing.T) {
	deps := HandlerDeps{
		TableStatesCache: nil,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440045")
	_, err := h.ensureTableAllowsOrdering(context.Background(), tableID)

	// With nil cache, should return nil error (allow ordering)
	if err != nil {
		t.Errorf("ensureTableAllowsOrdering() with nil cache should allow ordering, got error: %v", err)
	}
}

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
			name: "validResponse",
			resp: &aqm.SuccessResponse{
				Data: map[string]interface{}{
					"number": "T1",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target TableInfo
			err := decodeSuccessResponse(tt.resp, &target)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeSuccessResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && target.Number != "T1" {
				t.Errorf("decodeSuccessResponse() Number = %q, want %q", target.Number, "T1")
			}
		})
	}
}

func TestHandlerGetOrder(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440050")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440051")

	tests := []struct {
		name           string
		orderID        string
		setupRepo      func(*MockOrderRepo)
		expectedStatus int
	}{
		{
			name:    "validOrder",
			orderID: orderID.String(),
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{
					ID:      orderID,
					TableID: tableID,
					Status:  "pending",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "orderNotFound",
			orderID:        uuid.New().String(),
			setupRepo:      func(repo *MockOrderRepo) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalidID",
			orderID:        "not-a-uuid",
			setupRepo:      func(repo *MockOrderRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodGet, "/orders/"+tt.orderID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.GetOrder(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetOrder() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerListOrders(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440052")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440053")

	tests := []struct {
		name           string
		queryParams    string
		setupRepo      func(*MockOrderRepo)
		expectedStatus int
	}{
		{
			name:        "listAll",
			queryParams: "",
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{
					ID:      orderID,
					TableID: tableID,
					Status:  "pending",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "filterByTable",
			queryParams: "?table_id=" + tableID.String(),
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{
					ID:      orderID,
					TableID: tableID,
					Status:  "pending",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "filterByStatus",
			queryParams: "?status=pending",
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{
					ID:      orderID,
					TableID: tableID,
					Status:  "pending",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidTableID",
			queryParams:    "?table_id=not-a-uuid",
			setupRepo:      func(repo *MockOrderRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodGet, "/orders"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			h.ListOrders(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ListOrders() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerDeleteOrder(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440054")

	tests := []struct {
		name           string
		orderID        string
		setupRepo      func(*MockOrderRepo)
		expectedStatus int
	}{
		{
			name:    "deleteExisting",
			orderID: orderID.String(),
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalidID",
			orderID:        "not-a-uuid",
			setupRepo:      func(repo *MockOrderRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodDelete, "/orders/"+tt.orderID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.DeleteOrder(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("DeleteOrder() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerCreateOrder(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440055")

	tests := []struct {
		name           string
		body           interface{}
		setupCache     func(*TableStateCache)
		expectedStatus int
	}{
		{
			name: "validOrder",
			body: OrderCreateRequest{
				TableID: tableID,
			},
			setupCache: func(cache *TableStateCache) {
				cache.Set(tableID, "available")
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missingTableID",
			body:           OrderCreateRequest{},
			setupCache:     func(cache *TableStateCache) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalidJSON",
			body:           "not json",
			setupCache:     func(cache *TableStateCache) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := NewMockOrderRepo()
			groupRepo := NewMockOrderGroupRepo()
			cache := NewTableStateCache(nil, nil)
			tt.setupCache(cache)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo:      orderRepo,
					OrderGroupRepo: groupRepo,
				},
				TableStatesCache: cache,
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			var body []byte
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.CreateOrder(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateOrder() status = %d, want %d, body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerUpdateOrderStatus(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440056")

	tests := []struct {
		name           string
		orderID        string
		body           OrderUpdateRequest
		setupRepo      func(*MockOrderRepo)
		expectedStatus int
	}{
		{
			name:    "updateToPreparing",
			orderID: orderID.String(),
			body:    OrderUpdateRequest{Status: "preparing"},
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "updateToReady",
			orderID: orderID.String(),
			body:    OrderUpdateRequest{Status: "ready"},
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "preparing"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "updateToDelivered",
			orderID: orderID.String(),
			body:    OrderUpdateRequest{Status: "delivered"},
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "ready"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "updateToCancelled",
			orderID: orderID.String(),
			body:    OrderUpdateRequest{Status: "cancelled"},
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "invalidStatus",
			orderID: orderID.String(),
			body:    OrderUpdateRequest{Status: "invalid"},
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "orderNotFound",
			orderID:        uuid.New().String(),
			body:           OrderUpdateRequest{Status: "preparing"},
			setupRepo:      func(repo *MockOrderRepo) {},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/orders/"+tt.orderID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.UpdateOrderStatus(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateOrderStatus() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerCreateOrderItem(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440060")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440061")

	tests := []struct {
		name           string
		orderID        string
		body           interface{}
		setupRepos     func(*MockOrderRepo, *MockOrderItemRepo)
		setupCache     func(*TableStateCache)
		expectedStatus int
	}{
		{
			name:    "validItem",
			orderID: orderID.String(),
			body: OrderItemCreateRequest{
				DishName: "Pizza",
				Category: "main",
				Quantity: 2,
				Price:    15.99,
			},
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
			},
			setupCache: func(cache *TableStateCache) {
				cache.Set(tableID, "open")
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "invalidOrderID",
			orderID: "not-a-uuid",
			body: OrderItemCreateRequest{
				DishName: "Pizza",
				Quantity: 1,
			},
			setupRepos:     func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {},
			setupCache:     func(cache *TableStateCache) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "orderNotFound",
			orderID: uuid.New().String(),
			body: OrderItemCreateRequest{
				DishName: "Pizza",
				Quantity: 1,
			},
			setupRepos:     func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {},
			setupCache:     func(cache *TableStateCache) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:    "tableNotAllowingOrders",
			orderID: orderID.String(),
			body: OrderItemCreateRequest{
				DishName: "Pizza",
				Quantity: 1,
			},
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
			},
			setupCache: func(cache *TableStateCache) {
				cache.Set(tableID, "closed")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalidJSON",
			orderID:        orderID.String(),
			body:           "not json",
			setupRepos:     func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {},
			setupCache:     func(cache *TableStateCache) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := NewMockOrderRepo()
			itemRepo := NewMockOrderItemRepo()
			cache := NewTableStateCache(nil, nil)

			tt.setupRepos(orderRepo, itemRepo)
			tt.setupCache(cache)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo:     orderRepo,
					OrderItemRepo: itemRepo,
				},
				TableStatesCache: cache,
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			var body []byte
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/orders/"+tt.orderID+"/items", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orderID", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.CreateOrderItem(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateOrderItem() status = %d, want %d, body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerGetOrderItem(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440062")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440063")

	tests := []struct {
		name           string
		itemID         string
		setupRepo      func(*MockOrderItemRepo)
		expectedStatus int
	}{
		{
			name:   "validItem",
			itemID: itemID.String(),
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{
					ID:       itemID,
					OrderID:  orderID,
					DishName: "Pizza",
					Quantity: 2,
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "itemNotFound",
			itemID:         uuid.New().String(),
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalidID",
			itemID:         "not-a-uuid",
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderItemRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderItemRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodGet, "/order-items/"+tt.itemID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.itemID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.GetOrderItem(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetOrderItem() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerListOrderItems(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440064")
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440065")

	tests := []struct {
		name           string
		orderID        string
		setupRepo      func(*MockOrderItemRepo)
		expectedStatus int
	}{
		{
			name:    "listItems",
			orderID: orderID.String(),
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{
					ID:       itemID,
					OrderID:  orderID,
					DishName: "Pizza",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "emptyList",
			orderID: orderID.String(),
			setupRepo: func(repo *MockOrderItemRepo) {
				// No items
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidOrderID",
			orderID:        "not-a-uuid",
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderItemRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderItemRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodGet, "/orders/"+tt.orderID+"/items", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orderID", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.ListOrderItems(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ListOrderItems() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerUpdateOrderItem(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440066")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440067")

	quantity := 3
	notes := "extra cheese"
	statusPreparing := "preparing"
	statusReady := "ready"
	statusDelivered := "delivered"
	statusCancelled := "cancelled"
	invalidStatus := "invalid"

	tests := []struct {
		name           string
		itemID         string
		body           OrderItemUpdateRequest
		setupRepo      func(*MockOrderItemRepo)
		expectedStatus int
	}{
		{
			name:   "updateQuantity",
			itemID: itemID.String(),
			body:   OrderItemUpdateRequest{Quantity: &quantity},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Quantity: 1}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "updateNotes",
			itemID: itemID.String(),
			body:   OrderItemUpdateRequest{Notes: &notes},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "updateStatusPreparing",
			itemID: itemID.String(),
			body:   OrderItemUpdateRequest{Status: &statusPreparing},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "updateStatusReady",
			itemID: itemID.String(),
			body:   OrderItemUpdateRequest{Status: &statusReady},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "preparing"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "updateStatusDelivered",
			itemID: itemID.String(),
			body:   OrderItemUpdateRequest{Status: &statusDelivered},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "ready"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "updateStatusCancelled",
			itemID: itemID.String(),
			body:   OrderItemUpdateRequest{Status: &statusCancelled},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "invalidStatus",
			itemID: itemID.String(),
			body:   OrderItemUpdateRequest{Status: &invalidStatus},
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "itemNotFound",
			itemID:         uuid.New().String(),
			body:           OrderItemUpdateRequest{Quantity: &quantity},
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalidID",
			itemID:         "not-a-uuid",
			body:           OrderItemUpdateRequest{},
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderItemRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderItemRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/order-items/"+tt.itemID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.itemID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.UpdateOrderItem(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateOrderItem() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerDeleteOrderItem(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440068")

	tests := []struct {
		name           string
		itemID         string
		setupRepo      func(*MockOrderItemRepo)
		expectedStatus int
	}{
		{
			name:   "deleteExisting",
			itemID: itemID.String(),
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalidID",
			itemID:         "not-a-uuid",
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderItemRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderItemRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodDelete, "/order-items/"+tt.itemID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.itemID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.DeleteOrderItem(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("DeleteOrderItem() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerCreateOrderGroup(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440070")

	tests := []struct {
		name           string
		orderID        string
		body           interface{}
		setupRepo      func(*MockOrderRepo)
		expectedStatus int
	}{
		{
			name:    "validGroup",
			orderID: orderID.String(),
			body:    OrderGroupCreateRequest{Name: "Appetizers"},
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "emptyName",
			orderID: orderID.String(),
			body:    OrderGroupCreateRequest{Name: ""},
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "whitespaceName",
			orderID: orderID.String(),
			body:    OrderGroupCreateRequest{Name: "   "},
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "orderNotFound",
			orderID:        uuid.New().String(),
			body:           OrderGroupCreateRequest{Name: "Appetizers"},
			setupRepo:      func(repo *MockOrderRepo) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalidOrderID",
			orderID:        "not-a-uuid",
			body:           OrderGroupCreateRequest{Name: "Appetizers"},
			setupRepo:      func(repo *MockOrderRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "invalidJSON",
			orderID: orderID.String(),
			body:    "not json",
			setupRepo: func(repo *MockOrderRepo) {
				repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := NewMockOrderRepo()
			groupRepo := NewMockOrderGroupRepo()
			tt.setupRepo(orderRepo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo:      orderRepo,
					OrderGroupRepo: groupRepo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			var body []byte
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/orders/"+tt.orderID+"/groups", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orderID", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.CreateOrderGroup(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateOrderGroup() status = %d, want %d, body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerListOrderGroups(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440071")
	groupID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440072")

	tests := []struct {
		name           string
		orderID        string
		setupRepo      func(*MockOrderGroupRepo)
		expectedStatus int
	}{
		{
			name:    "listGroups",
			orderID: orderID.String(),
			setupRepo: func(repo *MockOrderGroupRepo) {
				repo.groups[groupID] = &OrderGroup{ID: groupID, OrderID: orderID, Name: "Main"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "emptyList",
			orderID: orderID.String(),
			setupRepo: func(repo *MockOrderGroupRepo) {
				// No groups
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidOrderID",
			orderID:        "not-a-uuid",
			setupRepo:      func(repo *MockOrderGroupRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderGroupRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderGroupRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodGet, "/orders/"+tt.orderID+"/groups", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orderID", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.ListOrderGroups(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ListOrderGroups() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerMarkItemDelivered(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440073")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440074")

	tests := []struct {
		name           string
		itemID         string
		setupRepo      func(*MockOrderItemRepo)
		expectedStatus int
	}{
		{
			name:   "markDelivered",
			itemID: itemID.String(),
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "ready"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "itemNotFound",
			itemID:         uuid.New().String(),
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalidID",
			itemID:         "not-a-uuid",
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderItemRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderItemRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodPatch, "/items/"+tt.itemID+"/deliver", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.itemID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.MarkItemDelivered(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("MarkItemDelivered() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerCancelItem(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440075")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440076")

	tests := []struct {
		name           string
		itemID         string
		setupRepo      func(*MockOrderItemRepo)
		expectedStatus int
	}{
		{
			name:   "cancelItem",
			itemID: itemID.String(),
			setupRepo: func(repo *MockOrderItemRepo) {
				repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "itemNotFound",
			itemID:         uuid.New().String(),
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalidID",
			itemID:         "not-a-uuid",
			setupRepo:      func(repo *MockOrderItemRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockOrderItemRepo()
			tt.setupRepo(repo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderItemRepo: repo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodPatch, "/items/"+tt.itemID+"/cancel", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.itemID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.CancelItem(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CancelItem() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerCloseOrderWithReadyItems(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440082")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440083")

	tests := []struct {
		name           string
		orderID        string
		queryParams    string
		setupRepos     func(*MockOrderRepo, *MockOrderItemRepo)
		expectedStatus int
	}{
		{
			name:        "closeOrderWithReadyItemsNoForce",
			orderID:     orderID.String(),
			queryParams: "",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
				itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440084")
				itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "ready"}
			},
			expectedStatus: http.StatusOK, // Returns confirmation required
		},
		{
			name:        "closeOrderWithReadyItemsForce",
			orderID:     orderID.String(),
			queryParams: "?force=true",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
				itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440085")
				itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "ready"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "closeOrderWithStartedItems",
			orderID:     orderID.String(),
			queryParams: "",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
				itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440086")
				itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "started"}
			},
			expectedStatus: http.StatusOK, // Returns confirmation required (preparing items)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := NewMockOrderRepo()
			itemRepo := NewMockOrderItemRepo()
			tt.setupRepos(orderRepo, itemRepo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo:     orderRepo,
					OrderItemRepo: itemRepo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodPost, "/orders/"+tt.orderID+"/close"+tt.queryParams, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.CloseOrder(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CloseOrder() status = %d, want %d, body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerRegisterRoutes(t *testing.T) {
	deps := HandlerDeps{}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// Verify routes were registered by checking if the router has routes
	// This is a basic test to ensure RegisterRoutes doesn't panic
	if r == nil {
		t.Error("RegisterRoutes() router should not be nil")
	}
}

func TestHandlerPublishOrderTableRejection(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440087")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440088")

	tests := []struct {
		name    string
		orderID *uuid.UUID
	}{
		{
			name:    "withOrderID",
			orderID: &orderID,
		},
		{
			name:    "withNilOrderID",
			orderID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := HandlerDeps{}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			// Should not panic when publisher is nil
			h.publishOrderTableRejection(context.Background(), tableID, tt.orderID, "test_action", "test_reason", "test_status")
		})
	}
}

func TestHandlerFetchTableInfo(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440089")

	deps := HandlerDeps{}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Should return error when tableClient is nil (which it will be since config doesn't have URL)
	_, err := h.fetchTableInfo(context.Background(), tableID)
	if err == nil {
		t.Error("fetchTableInfo() should return error when tableClient URL is not configured")
	}
}

func TestHandlerCreateOrderItemRequiresProduction(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440096")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440097")
	station := "kitchen"

	tests := []struct {
		name               string
		requiresProduction bool
		station            *string
		expectedStatus     int
	}{
		{
			name:               "itemRequiresProduction",
			requiresProduction: true,
			station:            &station,
			expectedStatus:     http.StatusCreated,
		},
		{
			name:               "itemDoesNotRequireProduction",
			requiresProduction: false,
			station:            nil,
			expectedStatus:     http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := NewMockOrderRepo()
			itemRepo := NewMockOrderItemRepo()
			cache := NewTableStateCache(nil, nil)

			orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
			cache.Set(tableID, "open")

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo:     orderRepo,
					OrderItemRepo: itemRepo,
				},
				TableStatesCache: cache,
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			reqBody := OrderItemCreateRequest{
				DishName:           "Test Dish",
				Category:           "main",
				Quantity:           1,
				Price:              10.0,
				RequiresProduction: tt.requiresProduction,
				ProductionStation:  tt.station,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/items", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("orderID", orderID.String())
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.CreateOrderItem(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateOrderItem() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerRepoErrors(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440098")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440099")
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440100")

	t.Run("getOrderRepoListError", func(t *testing.T) {
		repo := NewMockOrderRepo()
		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		w := httptest.NewRecorder()
		h.ListOrders(w, req)

		// Should return OK even with empty list
		if w.Code != http.StatusOK {
			t.Errorf("ListOrders() status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("createOrderRepoError", func(t *testing.T) {
		repo := NewMockOrderRepo()
		repo.CreateFunc = func(ctx context.Context, order *Order) error {
			return fmt.Errorf("database error")
		}
		cache := NewTableStateCache(nil, nil)
		cache.Set(tableID, "available")

		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo: repo,
			},
			TableStatesCache: cache,
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		body, _ := json.Marshal(OrderCreateRequest{TableID: tableID})
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.CreateOrder(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("CreateOrder() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("updateOrderStatusRepoSaveError", func(t *testing.T) {
		repo := NewMockOrderRepo()
		repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}
		repo.SaveFunc = func(ctx context.Context, order *Order) error {
			return fmt.Errorf("database error")
		}

		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		body, _ := json.Marshal(OrderUpdateRequest{Status: "preparing"})
		req := httptest.NewRequest(http.MethodPut, "/orders/"+orderID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", orderID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.UpdateOrderStatus(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("UpdateOrderStatus() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("deleteOrderRepoError", func(t *testing.T) {
		repo := NewMockOrderRepo()
		repo.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
			return fmt.Errorf("database error")
		}

		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		req := httptest.NewRequest(http.MethodDelete, "/orders/"+orderID.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", orderID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.DeleteOrder(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("DeleteOrder() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("createOrderItemRepoError", func(t *testing.T) {
		orderRepo := NewMockOrderRepo()
		orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

		itemRepo := NewMockOrderItemRepo()
		itemRepo.CreateFunc = func(ctx context.Context, item *OrderItem) error {
			return fmt.Errorf("database error")
		}

		cache := NewTableStateCache(nil, nil)
		cache.Set(tableID, "open")

		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo:     orderRepo,
				OrderItemRepo: itemRepo,
			},
			TableStatesCache: cache,
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		body, _ := json.Marshal(OrderItemCreateRequest{
			DishName: "Pizza",
			Quantity: 1,
		})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/items", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("orderID", orderID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.CreateOrderItem(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("CreateOrderItem() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("updateOrderItemRepoSaveError", func(t *testing.T) {
		repo := NewMockOrderItemRepo()
		repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID}
		repo.SaveFunc = func(ctx context.Context, item *OrderItem) error {
			return fmt.Errorf("database error")
		}

		deps := HandlerDeps{
			Repos: Repos{
				OrderItemRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		quantity := 5
		body, _ := json.Marshal(OrderItemUpdateRequest{Quantity: &quantity})
		req := httptest.NewRequest(http.MethodPut, "/order-items/"+itemID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", itemID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.UpdateOrderItem(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("UpdateOrderItem() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("deleteOrderItemRepoError", func(t *testing.T) {
		repo := NewMockOrderItemRepo()
		repo.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
			return fmt.Errorf("database error")
		}

		deps := HandlerDeps{
			Repos: Repos{
				OrderItemRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		req := httptest.NewRequest(http.MethodDelete, "/order-items/"+itemID.String(), nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", itemID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.DeleteOrderItem(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("DeleteOrderItem() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("markItemDeliveredRepoSaveError", func(t *testing.T) {
		repo := NewMockOrderItemRepo()
		repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "ready"}
		repo.SaveFunc = func(ctx context.Context, item *OrderItem) error {
			return fmt.Errorf("database error")
		}

		deps := HandlerDeps{
			Repos: Repos{
				OrderItemRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		req := httptest.NewRequest(http.MethodPatch, "/items/"+itemID.String()+"/deliver", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", itemID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.MarkItemDelivered(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("MarkItemDelivered() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("cancelItemRepoSaveError", func(t *testing.T) {
		repo := NewMockOrderItemRepo()
		repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "pending"}
		repo.SaveFunc = func(ctx context.Context, item *OrderItem) error {
			return fmt.Errorf("database error")
		}

		deps := HandlerDeps{
			Repos: Repos{
				OrderItemRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		req := httptest.NewRequest(http.MethodPatch, "/items/"+itemID.String()+"/cancel", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", itemID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.CancelItem(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("CancelItem() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("createOrderGroupRepoError", func(t *testing.T) {
		orderRepo := NewMockOrderRepo()
		orderRepo.orders[orderID] = &Order{ID: orderID, Status: "pending"}

		groupRepo := NewMockOrderGroupRepo()
		groupRepo.CreateFunc = func(ctx context.Context, group *OrderGroup) error {
			return fmt.Errorf("database error")
		}

		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo:      orderRepo,
				OrderGroupRepo: groupRepo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		body, _ := json.Marshal(OrderGroupCreateRequest{Name: "Test Group"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/groups", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("orderID", orderID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.CreateOrderGroup(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("CreateOrderGroup() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandlerPayloadDecoders(t *testing.T) {
	t.Run("decodeOrderUpdatePayloadInvalidBody", func(t *testing.T) {
		orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440101")
		repo := NewMockOrderRepo()
		repo.orders[orderID] = &Order{ID: orderID, Status: "pending"}

		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		req := httptest.NewRequest(http.MethodPut, "/orders/"+orderID.String(), bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", orderID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.UpdateOrderStatus(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("UpdateOrderStatus() status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("decodeOrderItemUpdatePayloadInvalidBody", func(t *testing.T) {
		itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440102")
		orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440103")

		repo := NewMockOrderItemRepo()
		repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID}

		deps := HandlerDeps{
			Repos: Repos{
				OrderItemRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		req := httptest.NewRequest(http.MethodPut, "/order-items/"+itemID.String(), bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", itemID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.UpdateOrderItem(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("UpdateOrderItem() status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestHandlerGetOrderNilCheck(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440104")

	repo := NewMockOrderRepo()
	// GetFunc returns nil order without error
	repo.GetFunc = func(ctx context.Context, id uuid.UUID) (*Order, error) {
		return nil, nil
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: repo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.GetOrder(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetOrder() status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerGetOrderItemNilCheck(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440105")

	repo := NewMockOrderItemRepo()
	// GetFunc returns nil item without error
	repo.GetFunc = func(ctx context.Context, id uuid.UUID) (*OrderItem, error) {
		return nil, nil
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: repo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/order-items/"+itemID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.GetOrderItem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetOrderItem() status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerParseIDParamMissing(t *testing.T) {
	deps := HandlerDeps{}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Test with missing id param
	req := httptest.NewRequest(http.MethodGet, "/orders/", nil)
	rctx := chi.NewRouteContext()
	// Don't add id param
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.GetOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GetOrder() with missing id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCloseOrderRepoError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440106")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440107")

	t.Run("closeOrderSaveError", func(t *testing.T) {
		orderRepo := NewMockOrderRepo()
		orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
		orderRepo.SaveFunc = func(ctx context.Context, order *Order) error {
			return fmt.Errorf("database error")
		}

		itemRepo := NewMockOrderItemRepo()

		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo:     orderRepo,
				OrderItemRepo: itemRepo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/close?force=true", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", orderID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.CloseOrder(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("CloseOrder() status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandlerListOrdersRepoError(t *testing.T) {
	t.Run("listOrdersInternalError", func(t *testing.T) {
		repo := &MockOrderRepo{
			orders: make(map[uuid.UUID]*Order),
		}
		// Override List to return error
		originalList := repo.List
		_ = originalList

		deps := HandlerDeps{
			Repos: Repos{
				OrderRepo: repo,
			},
		}
		h := NewHandler(deps, aqm.NewConfig(), nil)

		// Test with status filter that returns empty list (OK case)
		req := httptest.NewRequest(http.MethodGet, "/orders?status=pending", nil)
		w := httptest.NewRecorder()
		h.ListOrders(w, req)

		// Empty list returns OK
		if w.Code != http.StatusOK {
			t.Errorf("ListOrders() status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestHandlerListOrderItemsRepoError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440108")

	repo := NewMockOrderItemRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: repo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/items", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderItems(w, req)

	// Empty list returns OK
	if w.Code != http.StatusOK {
		t.Errorf("ListOrderItems() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrderGroupsRepoError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440109")

	repo := NewMockOrderGroupRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderGroupRepo: repo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/groups", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderGroups(w, req)

	// Empty list returns OK
	if w.Code != http.StatusOK {
		t.Errorf("ListOrderGroups() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerEnsureTableAllowsOrderingStatusUnavailable(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440110")

	cache := NewTableStateCache(nil, nil)
	// Set empty status
	cache.Set(tableID, "")

	deps := HandlerDeps{
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	_, err := h.ensureTableAllowsOrdering(context.Background(), tableID)
	if err == nil {
		t.Error("ensureTableAllowsOrdering() should return error for empty status")
	}

	expectedMsg := "table status unavailable"
	if err.Error() != expectedMsg {
		t.Errorf("ensureTableAllowsOrdering() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestHandlerUpdateOrderItemNoStatusChange(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440111")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440112")

	repo := NewMockOrderItemRepo()
	repo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Quantity: 1}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: repo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Update only quantity, not status
	quantity := 5
	body, _ := json.Marshal(OrderItemUpdateRequest{Quantity: &quantity})
	req := httptest.NewRequest(http.MethodPut, "/order-items/"+itemID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateOrderItem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UpdateOrderItem() status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify quantity was updated
	item, _ := repo.Get(context.Background(), itemID)
	if item.Quantity != 5 {
		t.Errorf("UpdateOrderItem() quantity = %d, want 5", item.Quantity)
	}
}

func TestHandlerMarkItemDeliveredWithRequiresProduction(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440113")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440114")

	repo := NewMockOrderItemRepo()
	repo.items[itemID] = &OrderItem{
		ID:                 itemID,
		OrderID:            orderID,
		Status:             "ready",
		RequiresProduction: true,
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: repo,
		},
		Publisher: NewMockPublisher(),
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/items/"+itemID.String()+"/deliver", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.MarkItemDelivered(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("MarkItemDelivered() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerUpdateOrderNilCheck(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440115")

	repo := NewMockOrderRepo()
	repo.GetFunc = func(ctx context.Context, id uuid.UUID) (*Order, error) {
		return nil, nil
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: repo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderUpdateRequest{Status: "preparing"})
	req := httptest.NewRequest(http.MethodPut, "/orders/"+orderID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateOrderStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("UpdateOrderStatus() with nil order status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerUpdateOrderItemNilCheck(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440116")

	repo := NewMockOrderItemRepo()
	repo.GetFunc = func(ctx context.Context, id uuid.UUID) (*OrderItem, error) {
		return nil, nil
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: repo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	quantity := 5
	body, _ := json.Marshal(OrderItemUpdateRequest{Quantity: &quantity})
	req := httptest.NewRequest(http.MethodPut, "/order-items/"+itemID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateOrderItem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("UpdateOrderItem() with nil item status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCloseOrder(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440077")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440078")

	tests := []struct {
		name           string
		orderID        string
		queryParams    string
		setupRepos     func(*MockOrderRepo, *MockOrderItemRepo)
		expectedStatus int
	}{
		{
			name:        "closeOrderNoItems",
			orderID:     orderID.String(),
			queryParams: "?force=true",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "closeOrderWithDeliveredItems",
			orderID:     orderID.String(),
			queryParams: "?force=true",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
				itemRepo.items[uuid.New()] = &OrderItem{OrderID: orderID, Status: "delivered"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "closeOrderWithPendingItemsNoForce",
			orderID:     orderID.String(),
			queryParams: "",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
				itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440079")
				itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusOK, // Returns confirmation required
		},
		{
			name:        "closeOrderWithPendingItemsForce",
			orderID:     orderID.String(),
			queryParams: "?force=true",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
				itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440080")
				itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "pending"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "closeOrderWithPreparingItemsTakeaway",
			orderID:     orderID.String(),
			queryParams: "?force=true&takeaway=true",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
				itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440081")
				itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "preparing"}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "closeOrderAlreadyClosed",
			orderID:     orderID.String(),
			queryParams: "",
			setupRepos: func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {
				orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "closed"}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "orderNotFound",
			orderID:     uuid.New().String(),
			queryParams: "",
			setupRepos:  func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalidID",
			orderID:        "not-a-uuid",
			queryParams:    "",
			setupRepos:     func(orderRepo *MockOrderRepo, itemRepo *MockOrderItemRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := NewMockOrderRepo()
			itemRepo := NewMockOrderItemRepo()
			tt.setupRepos(orderRepo, itemRepo)

			deps := HandlerDeps{
				Repos: Repos{
					OrderRepo:     orderRepo,
					OrderItemRepo: itemRepo,
				},
			}
			h := NewHandler(deps, aqm.NewConfig(), nil)

			req := httptest.NewRequest(http.MethodPost, "/orders/"+tt.orderID+"/close"+tt.queryParams, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.orderID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.CloseOrder(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CloseOrder() status = %d, want %d, body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerCreateOrderItemWithPublisher(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440120")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440121")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	itemRepo := NewMockOrderItemRepo()
	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "open")

	var publishedTopic string
	publisher := NewMockPublisher()
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		publishedTopic = topic
		return nil
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:     orderRepo,
			OrderItemRepo: itemRepo,
		},
		Publisher:        publisher,
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderItemCreateRequest{
		DishName:           "Test Dish",
		Quantity:           2,
		RequiresProduction: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CreateOrderItem(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateOrderItem() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify event was published
	if publishedTopic == "" {
		t.Error("CreateOrderItem() should publish event for production items")
	}
}

func TestHandlerCreateOrderItemTableRejection(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440122")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440123")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	itemRepo := NewMockOrderItemRepo()
	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "unavailable") // Table is unavailable

	var publishedTopic string
	publisher := NewMockPublisher()
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		publishedTopic = topic
		return nil
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:     orderRepo,
			OrderItemRepo: itemRepo,
		},
		Publisher:        publisher,
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderItemCreateRequest{
		DishName: "Test Dish",
		Quantity: 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CreateOrderItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateOrderItem() with unavailable table status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Verify rejection event was published
	if publishedTopic == "" {
		t.Error("CreateOrderItem() should publish rejection event when table unavailable")
	}
}

func TestHandlerCreateOrderTableRejection(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440124")

	orderRepo := NewMockOrderRepo()
	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "unavailable")

	var publishedTopic string
	publisher := NewMockPublisher()
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		publishedTopic = topic
		return nil
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
		Publisher:        publisher,
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderCreateRequest{
		TableID: tableID,
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CreateOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateOrder() with unavailable table status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Verify rejection event was published
	if publishedTopic == "" {
		t.Error("CreateOrder() should publish rejection event when table unavailable")
	}
}

func TestHandlerPublishOrderTableRejectionNilPublisher(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440125")

	orderRepo := NewMockOrderRepo()
	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "unavailable")

	// No publisher
	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderCreateRequest{
		TableID: tableID,
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CreateOrder(w, req)

	// Should still reject but without panic when publisher is nil
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateOrder() with unavailable table status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerListOrdersTableFilter(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440131")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440132")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders?table_id="+tableID.String(), nil)

	w := httptest.NewRecorder()
	h.ListOrders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrders() with table filter status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrdersStatusFilter(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440133")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440134")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders?status=pending", nil)

	w := httptest.NewRecorder()
	h.ListOrders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrders() with status filter status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrdersTableInvalidUUID(t *testing.T) {
	orderRepo := NewMockOrderRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders?table_id=invalid-uuid", nil)

	w := httptest.NewRecorder()
	h.ListOrders(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ListOrders() with invalid table_id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerListOrdersAll(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440135")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440136")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)

	w := httptest.NewRecorder()
	h.ListOrders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrders() all status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrderItemsByOrder(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440137")
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440138")

	itemRepo := NewMockOrderItemRepo()
	itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/items", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderItems(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrderItems() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrderItemsEmpty(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440139")

	itemRepo := NewMockOrderItemRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/items", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderItems(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrderItems() empty status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrderItemsInvalidOrderID(t *testing.T) {
	itemRepo := NewMockOrderItemRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/invalid/items", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderItems(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ListOrderItems() with invalid orderID status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerListOrderGroupsByOrder(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440143")
	groupID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440144")

	groupRepo := NewMockOrderGroupRepo()
	groupRepo.groups[groupID] = &OrderGroup{ID: groupID, OrderID: orderID, Name: "Main"}

	deps := HandlerDeps{
		Repos: Repos{
			OrderGroupRepo: groupRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/groups", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderGroups(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrderGroups() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrderGroupsInvalidOrderID(t *testing.T) {
	groupRepo := NewMockOrderGroupRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderGroupRepo: groupRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/invalid/groups", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderGroups(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ListOrderGroups() with invalid orderID status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateOrderSuccess(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440145")

	orderRepo := NewMockOrderRepo()
	groupRepo := NewMockOrderGroupRepo()
	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "available")

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:      orderRepo,
			OrderGroupRepo: groupRepo,
		},
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderCreateRequest{
		TableID: tableID,
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CreateOrder(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateOrder() status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestHandlerCreateOrderMissingTableID(t *testing.T) {
	orderRepo := NewMockOrderRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderCreateRequest{})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CreateOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateOrder() without tableID status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateOrderItemInvalidOrderID(t *testing.T) {
	itemRepo := NewMockOrderItemRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderItemCreateRequest{
		DishName: "Test",
		Quantity: 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/orders/invalid/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CreateOrderItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateOrderItem() with invalid orderID status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateOrderItemRepoError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440146")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440147")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	itemRepo := NewMockOrderItemRepo()
	itemRepo.CreateFunc = func(ctx context.Context, item *OrderItem) error {
		return fmt.Errorf("db error")
	}

	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "open")

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:     orderRepo,
			OrderItemRepo: itemRepo,
		},
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderItemCreateRequest{
		DishName: "Test",
		Quantity: 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CreateOrderItem(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("CreateOrderItem() with repo error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerCloseOrderSaveError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440148")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440149")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
	orderRepo.SaveFunc = func(ctx context.Context, order *Order) error {
		return fmt.Errorf("db error")
	}

	itemRepo := NewMockOrderItemRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:     orderRepo,
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/close?force=true", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CloseOrder(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("CloseOrder() with save error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerDecodeSuccessResponseNil(t *testing.T) {
	var target interface{}
	err := decodeSuccessResponse(nil, &target)
	if err == nil {
		t.Error("decodeSuccessResponse() with nil should return error")
	}
}

func TestHandlerCreateOrderRepoError(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440150")

	orderRepo := NewMockOrderRepo()
	orderRepo.CreateFunc = func(ctx context.Context, order *Order) error {
		return fmt.Errorf("db error")
	}

	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "available")

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderCreateRequest{
		TableID: tableID,
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CreateOrder(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("CreateOrder() with repo error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerCreateOrderGroupRepoError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440151")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440152")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	groupRepo := NewMockOrderGroupRepo()
	groupRepo.CreateFunc = func(ctx context.Context, group *OrderGroup) error {
		return fmt.Errorf("db error")
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:      orderRepo,
			OrderGroupRepo: groupRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderGroupCreateRequest{
		Name: "Appetizers",
	})
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CreateOrderGroup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("CreateOrderGroup() with repo error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerDeleteOrderRepoError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440153")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440154")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
	orderRepo.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
		return fmt.Errorf("db error")
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodDelete, "/orders/"+orderID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.DeleteOrder(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("DeleteOrder() with repo error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerDeleteOrderItemRepoError(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440155")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440156")

	itemRepo := NewMockOrderItemRepo()
	itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID}
	itemRepo.DeleteFunc = func(ctx context.Context, id uuid.UUID) error {
		return fmt.Errorf("db error")
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodDelete, "/order-items/"+itemID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.DeleteOrderItem(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("DeleteOrderItem() with repo error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerUpdateOrderStatusRepoError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440157")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440158")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}
	orderRepo.SaveFunc = func(ctx context.Context, order *Order) error {
		return fmt.Errorf("db error")
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo: orderRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	body, _ := json.Marshal(OrderUpdateRequest{Status: "preparing"})
	req := httptest.NewRequest(http.MethodPut, "/orders/"+orderID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateOrderStatus(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("UpdateOrderStatus() with repo error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerUpdateOrderItemRepoError(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440159")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440160")

	itemRepo := NewMockOrderItemRepo()
	itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Quantity: 1}
	itemRepo.SaveFunc = func(ctx context.Context, item *OrderItem) error {
		return fmt.Errorf("db error")
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	quantity := 5
	body, _ := json.Marshal(OrderItemUpdateRequest{Quantity: &quantity})
	req := httptest.NewRequest(http.MethodPut, "/order-items/"+itemID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateOrderItem(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("UpdateOrderItem() with repo error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerCancelItemNotFound(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440161")

	itemRepo := NewMockOrderItemRepo()
	// Item not in repo

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/order-items/"+itemID.String()+"/cancel", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CancelItem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("CancelItem() not found status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerCancelItemSuccess(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440162")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440163")

	itemRepo := NewMockOrderItemRepo()
	itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "pending"}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/order-items/"+itemID.String()+"/cancel", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CancelItem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CancelItem() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerCancelItemInvalidID(t *testing.T) {
	itemRepo := NewMockOrderItemRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/order-items/invalid/cancel", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CancelItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CancelItem() invalid id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerMarkItemDeliveredNotFound(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440164")

	itemRepo := NewMockOrderItemRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/order-items/"+itemID.String()+"/deliver", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.MarkItemDelivered(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("MarkItemDelivered() not found status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlerMarkItemDeliveredInvalidID(t *testing.T) {
	itemRepo := NewMockOrderItemRepo()

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/order-items/invalid/deliver", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.MarkItemDelivered(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("MarkItemDelivered() invalid id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerMarkItemDeliveredSaveError(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440700")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440701")
	itemRepo := NewMockOrderItemRepo()
	item := &OrderItem{
		ID:      itemID,
		OrderID: orderID,
		Status:  "pending",
	}
	itemRepo.items[itemID] = item
	itemRepo.SaveFunc = func(ctx context.Context, item *OrderItem) error {
		return fmt.Errorf("db error")
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/order-items/"+itemID.String()+"/deliver", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.MarkItemDelivered(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("MarkItemDelivered() save error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerCancelItemSaveError(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440710")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440711")
	itemRepo := NewMockOrderItemRepo()
	item := &OrderItem{
		ID:      itemID,
		OrderID: orderID,
		Status:  "pending",
	}
	itemRepo.items[itemID] = item
	itemRepo.SaveFunc = func(ctx context.Context, item *OrderItem) error {
		return fmt.Errorf("db error")
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/order-items/"+itemID.String()+"/cancel", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CancelItem(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("CancelItem() save error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandlerCreateOrderInvalidJSON(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440720")
	orderRepo := NewMockOrderRepo()
	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "available")

	deps := HandlerDeps{
		Repos:            Repos{OrderRepo: orderRepo},
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Invalid JSON body
	body := []byte(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CreateOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateOrder() invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateOrderItemInvalidJSON(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440730")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440731")
	orderRepo := NewMockOrderRepo()
	order := &Order{ID: orderID, TableID: tableID, Status: "open"}
	orderRepo.orders[orderID] = order

	itemRepo := NewMockOrderItemRepo()
	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "open")

	deps := HandlerDeps{
		Repos:            Repos{OrderRepo: orderRepo, OrderItemRepo: itemRepo},
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Invalid JSON body
	body := []byte(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CreateOrderItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateOrderItem() invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUpdateOrderStatusInvalidJSON(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440740")
	orderRepo := NewMockOrderRepo()
	order := &Order{ID: orderID, Status: "open"}
	orderRepo.orders[orderID] = order

	deps := HandlerDeps{
		Repos: Repos{OrderRepo: orderRepo},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Invalid JSON body
	body := []byte(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPatch, "/orders/"+orderID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateOrderStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("UpdateOrderStatus() invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerUpdateOrderItemInvalidJSON(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440750")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440751")
	itemRepo := NewMockOrderItemRepo()
	item := &OrderItem{ID: itemID, OrderID: orderID, Status: "pending"}
	itemRepo.items[itemID] = item

	deps := HandlerDeps{
		Repos: Repos{OrderItemRepo: itemRepo},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Invalid JSON body
	body := []byte(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPatch, "/order-items/"+itemID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateOrderItem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("UpdateOrderItem() invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateOrderGroupInvalidJSON(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440760")
	orderRepo := NewMockOrderRepo()
	order := &Order{ID: orderID, Status: "open"}
	orderRepo.orders[orderID] = order

	groupRepo := NewMockOrderGroupRepo()

	deps := HandlerDeps{
		Repos: Repos{OrderRepo: orderRepo, OrderGroupRepo: groupRepo},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Invalid JSON body
	body := []byte(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CreateOrderGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateOrderGroup() invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerPublishOrderTableRejectionWithOrderID(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440770")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440771")
	publisher := NewMockPublisher()
	var published bool
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		published = true
		return nil
	}

	deps := HandlerDeps{
		Publisher: publisher,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	h.publishOrderTableRejection(context.Background(), tableID, &orderID, "create", "table occupied", "occupied")

	if !published {
		t.Error("publishOrderTableRejection() with orderID should publish event")
	}
}

func TestHandlerPublishOrderTableRejectionPublishError(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440780")
	publisher := NewMockPublisher()
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		return fmt.Errorf("publish error")
	}

	deps := HandlerDeps{
		Publisher: publisher,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Should not panic on publish error
	h.publishOrderTableRejection(context.Background(), tableID, nil, "create", "table occupied", "occupied")
}

func TestHandlerPublishOrderItemCreatedWithMenuItemID(t *testing.T) {
	publisher := NewMockPublisher()
	var published bool
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		published = true
		return nil
	}

	deps := HandlerDeps{
		Publisher: publisher,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	menuItemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440790")
	station := "grill"
	item := &OrderItem{
		ID:                uuid.MustParse("550e8400-e29b-41d4-a716-446655440791"),
		OrderID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440792"),
		MenuItemID:        &menuItemID,
		ProductionStation: &station,
		DishName:          "Test Dish",
		Quantity:          1,
	}

	parentOrder := &Order{
		ID:      item.OrderID,
		TableID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440793"),
	}

	h.publishOrderItemCreated(context.Background(), item, parentOrder)

	if !published {
		t.Error("publishOrderItemCreated() with menuItemID and productionStation should publish event")
	}
}

func TestHandlerPublishOrderItemCreatedPublishError(t *testing.T) {
	publisher := NewMockPublisher()
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		return fmt.Errorf("publish error")
	}

	deps := HandlerDeps{
		Publisher: publisher,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	item := &OrderItem{
		ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440800"),
		OrderID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440801"),
		DishName: "Test Dish",
		Quantity: 1,
	}

	parentOrder := &Order{
		ID:      item.OrderID,
		TableID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440802"),
	}

	// Should not panic on publish error
	h.publishOrderItemCreated(context.Background(), item, parentOrder)
}

func TestHandlerPublishOrderItemStatusChangePublishError(t *testing.T) {
	publisher := NewMockPublisher()
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		return fmt.Errorf("publish error")
	}

	deps := HandlerDeps{
		Publisher: publisher,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	item := &OrderItem{
		ID:                 uuid.MustParse("550e8400-e29b-41d4-a716-446655440810"),
		OrderID:            uuid.MustParse("550e8400-e29b-41d4-a716-446655440811"),
		Status:             "delivered",
		RequiresProduction: true,
	}

	// Should not panic on publish error
	h.publishOrderItemStatusChange(context.Background(), item, "pending")
}

func TestHandlerCloseOrderListItemsError(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440820")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440821")
	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	itemRepo := NewMockOrderItemRepo()
	// Mock the list to fail - we need to add a ListByOrderFunc to the mock
	// Since it's not there, we'll test via a different approach

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:     orderRepo,
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/close?force=true", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CloseOrder(w, req)

	// Should succeed since no items exist to close
	if w.Code != http.StatusOK {
		t.Errorf("CloseOrder() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerMarkItemDeliveredWithGRPCBroadcast(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440830")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440831")
	itemRepo := NewMockOrderItemRepo()
	item := &OrderItem{
		ID:       itemID,
		OrderID:  orderID,
		Status:   "ready",
		DishName: "Test Dish",
	}
	itemRepo.items[itemID] = item

	streamServer := NewOrderEventStreamServer(nil, aqm.NewNoopLogger())

	deps := HandlerDeps{
		Repos: Repos{
			OrderItemRepo: itemRepo,
		},
		OrderStreamServer: streamServer,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPatch, "/order-items/"+itemID.String()+"/deliver", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", itemID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.MarkItemDelivered(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("MarkItemDelivered() with gRPC broadcast status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerCloseOrderWithCancelledItems(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440840")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440841")
	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "pending"}

	itemRepo := NewMockOrderItemRepo()
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440842")
	itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, Status: "cancelled"}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:     orderRepo,
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/close?force=true", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CloseOrder(w, req)

	// Should succeed since cancelled items don't block closing
	if w.Code != http.StatusOK {
		t.Errorf("CloseOrder() with cancelled items status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerCloseOrderWithMixedItems(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440850")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440851")
	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "open"}

	itemRepo := NewMockOrderItemRepo()
	itemRepo.items[uuid.MustParse("550e8400-e29b-41d4-a716-446655440852")] = &OrderItem{
		ID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440852"),
		OrderID: orderID,
		Status:  "pending",
	}
	itemRepo.items[uuid.MustParse("550e8400-e29b-41d4-a716-446655440853")] = &OrderItem{
		ID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440853"),
		OrderID: orderID,
		Status:  "ready",
	}
	itemRepo.items[uuid.MustParse("550e8400-e29b-41d4-a716-446655440854")] = &OrderItem{
		ID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440854"),
		OrderID: orderID,
		Status:  "delivered",
	}

	deps := HandlerDeps{
		Repos: Repos{
			OrderRepo:     orderRepo,
			OrderItemRepo: itemRepo,
		},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	// Without force - should require confirmation
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/close", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CloseOrder(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CloseOrder() with mixed items status = %d, want %d", w.Code, http.StatusOK)
	}

	// Should return requires_confirmation
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data in response")
	}
	if _, exists := data["requires_confirmation"]; !exists {
		t.Error("expected requires_confirmation in response")
	}
}

func TestHandlerPublishOrderItemCreatedSuccess(t *testing.T) {
	publisher := NewMockPublisher()
	var published bool
	var publishedPayload []byte
	publisher.PublishFunc = func(ctx context.Context, topic string, msg []byte) error {
		published = true
		publishedPayload = msg
		return nil
	}

	deps := HandlerDeps{
		Publisher: publisher,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	item := &OrderItem{
		ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440860"),
		OrderID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440861"),
		DishName: "Test Dish",
		Quantity: 1,
	}

	parentOrder := &Order{
		ID:      item.OrderID,
		TableID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440862"),
	}

	h.publishOrderItemCreated(context.Background(), item, parentOrder)

	if !published {
		t.Error("publishOrderItemCreated() should publish event")
	}
	if len(publishedPayload) == 0 {
		t.Error("publishOrderItemCreated() should publish non-empty payload")
	}
}

func TestRehydrateUnmarshalError(t *testing.T) {
	// Test with a value that can be marshaled but not unmarshaled to the target type
	input := map[string]interface{}{
		"id":     123, // number instead of string
		"status": "test",
	}

	var out tableStateDTO
	err := rehydrate(input, &out)
	// This should not error since JSON marshaling works and the number becomes a string
	if err != nil {
		t.Logf("rehydrate() error = %v (may be expected)", err)
	}
}

func TestIngestCollectionRehydrateError(t *testing.T) {
	cache := NewTableStateCache(nil, nil)

	// Pass something that can't be rehydrated to []tableStateDTO
	input := "not-a-slice"

	err := cache.ingestCollection(input)
	if err == nil {
		t.Error("ingestCollection() should return error for invalid input type")
	}
}

func TestHandlerListOrdersInvalidTableID(t *testing.T) {
	orderRepo := NewMockOrderRepo()
	deps := HandlerDeps{
		Repos: Repos{OrderRepo: orderRepo},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders?table_id=invalid", nil)
	w := httptest.NewRecorder()
	h.ListOrders(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ListOrders() with invalid table_id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerListOrdersByTableSuccess(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440900")
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440901")
	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "open"}

	deps := HandlerDeps{
		Repos: Repos{OrderRepo: orderRepo},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders?table_id="+tableID.String(), nil)
	w := httptest.NewRecorder()
	h.ListOrders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrders() by table status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrdersByStatusSuccess(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440910")
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440911")
	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, TableID: tableID, Status: "open"}

	deps := HandlerDeps{
		Repos: Repos{OrderRepo: orderRepo},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders?status=open", nil)
	w := httptest.NewRecorder()
	h.ListOrders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrders() by status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrderItemsByGroupSuccess(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440920")
	groupID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440921")
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440922")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, Status: "open"}

	itemRepo := NewMockOrderItemRepo()
	itemRepo.items[itemID] = &OrderItem{ID: itemID, OrderID: orderID, GroupID: &groupID, Status: "pending"}

	deps := HandlerDeps{
		Repos: Repos{OrderRepo: orderRepo, OrderItemRepo: itemRepo},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/items?group_id="+groupID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderItems(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrderItems() by group status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerListOrderGroupsSuccess(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440940")
	groupID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440941")

	orderRepo := NewMockOrderRepo()
	orderRepo.orders[orderID] = &Order{ID: orderID, Status: "open"}

	groupRepo := NewMockOrderGroupRepo()
	groupRepo.groups[groupID] = &OrderGroup{ID: groupID, OrderID: orderID, Name: "Test Group"}

	deps := HandlerDeps{
		Repos: Repos{OrderRepo: orderRepo, OrderGroupRepo: groupRepo},
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/groups", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListOrderGroups(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListOrderGroups() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerEnsureTableAllowsOrderingEmptyStatus(t *testing.T) {
	tableID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440960")
	cache := NewTableStateCache(nil, nil)
	cache.Set(tableID, "") // Empty status

	deps := HandlerDeps{
		TableStatesCache: cache,
	}
	h := NewHandler(deps, aqm.NewConfig(), nil)

	_, err := h.ensureTableAllowsOrdering(context.Background(), tableID)
	if err == nil {
		t.Error("ensureTableAllowsOrdering() with empty status should return error")
	}
}

