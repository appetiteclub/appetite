package kitchen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/appetiteclub/appetite/pkg/enums/kitchenstatus"
	"github.com/appetiteclub/apt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name   string
		deps   HandlerDeps
		config *apt.Config
		logger apt.Logger
	}{
		{
			name: "withAllDependencies",
			deps: HandlerDeps{
				Repo:      NewMockTicketRepository(),
				Cache:     NewTicketStateCache(nil, nil, nil),
				Publisher: NewMockPublisher(),
			},
			config: apt.NewConfig(),
			logger: apt.NewNoopLogger(),
		},
		{
			name:   "withNilLogger",
			deps:   HandlerDeps{},
			config: apt.NewConfig(),
			logger: nil,
		},
		{
			name:   "withEmptyDeps",
			deps:   HandlerDeps{},
			config: nil,
			logger: apt.NewNoopLogger(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.deps, tt.config, tt.logger)
			if h == nil {
				t.Error("NewHandler() returned nil")
			}
		})
	}
}

func TestHandlerRegisterRoutes(t *testing.T) {
	h := NewHandler(HandlerDeps{}, nil, apt.NewNoopLogger())
	r := chi.NewRouter()

	// Should not panic
	h.RegisterRoutes(r)
}

func TestHandlerListTickets(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		setupCache     func(*TicketStateCache)
		setupRepo      func(*MockTicketRepository)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:  "listAllFromCache",
			query: "",
			setupCache: func(c *TicketStateCache) {
				c.Set(&Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"})
				c.Set(&Ticket{ID: uuid.New(), Station: "bar", Status: "started"})
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:  "filterByStationFromCache",
			query: "?station=kitchen",
			setupCache: func(c *TicketStateCache) {
				c.Set(&Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"})
				c.Set(&Ticket{ID: uuid.New(), Station: "bar", Status: "started"})
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:  "filterByStatusFromCache",
			query: "?status=created",
			setupCache: func(c *TicketStateCache) {
				c.Set(&Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"})
				c.Set(&Ticket{ID: uuid.New(), Station: "bar", Status: "started"})
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:  "filterByStationAndStatusFromCache",
			query: "?station=kitchen&status=created",
			setupCache: func(c *TicketStateCache) {
				c.Set(&Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"})
				c.Set(&Ticket{ID: uuid.New(), Station: "kitchen", Status: "started"})
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:  "filterByOrderIDFromRepo",
			query: "?order_id=" + uuid.New().String(),
			setupRepo: func(r *MockTicketRepository) {
				r.ListFunc = func(ctx context.Context, filter TicketFilter) ([]Ticket, error) {
					return []Ticket{{ID: uuid.New(), Station: "kitchen"}}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:  "filterByOrderItemIDFromRepo",
			query: "?order_item_id=" + uuid.New().String(),
			setupRepo: func(r *MockTicketRepository) {
				r.ListFunc = func(ctx context.Context, filter TicketFilter) ([]Ticket, error) {
					return []Ticket{}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "invalidOrderID",
			query:          "?order_id=invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalidOrderItemID",
			query:          "?order_item_id=invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "repoListError",
			query: "?order_id=" + uuid.New().String(),
			setupRepo: func(r *MockTicketRepository) {
				r.ListFunc = func(ctx context.Context, filter TicketFilter) ([]Ticket, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupCache != nil {
				tt.setupCache(cache)
			}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			req := httptest.NewRequest(http.MethodGet, "/tickets"+tt.query, nil)
			w := httptest.NewRecorder()

			h.ListTickets(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ListTickets() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK && tt.expectedCount >= 0 {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				data, ok := resp["data"].(map[string]interface{})
				if !ok {
					t.Fatalf("Response does not contain data object: %s", w.Body.String())
				}
				tickets, ok := data["tickets"].([]interface{})
				if !ok {
					t.Fatalf("Response does not contain tickets array: %s", w.Body.String())
				}
				if len(tickets) != tt.expectedCount {
					t.Errorf("tickets count = %d, want %d", len(tickets), tt.expectedCount)
				}
			}
		})
	}
}

func TestHandlerListTicketsNilCache(t *testing.T) {
	repo := NewMockTicketRepository()
	repo.ListFunc = func(ctx context.Context, filter TicketFilter) ([]Ticket, error) {
		return []Ticket{{ID: uuid.New(), Station: "kitchen"}}, nil
	}

	deps := HandlerDeps{Repo: repo, Cache: nil, Publisher: NewMockPublisher()}
	h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

	req := httptest.NewRequest(http.MethodGet, "/tickets", nil)
	w := httptest.NewRecorder()

	h.ListTickets(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListTickets() with nil cache status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerGetTicket(t *testing.T) {
	ticketID := uuid.New()

	tests := []struct {
		name           string
		ticketID       string
		setupRepo      func(*MockTicketRepository)
		expectedStatus int
	}{
		{
			name:     "success",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidID",
			ticketID:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "notFound",
			ticketID: uuid.New().String(),
			setupRepo: func(r *MockTicketRepository) {
				// No ticket added
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "repoError",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.FindByIDFunc = func(ctx context.Context, id TicketID) (*Ticket, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			deps := HandlerDeps{Repo: repo, Cache: nil, Publisher: NewMockPublisher()}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			r := chi.NewRouter()
			r.Get("/tickets/{id}", h.GetTicket)

			req := httptest.NewRequest(http.MethodGet, "/tickets/"+tt.ticketID, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetTicket() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerAcceptTicket(t *testing.T) {
	ticketID := uuid.New()

	tests := []struct {
		name           string
		ticketID       string
		setupRepo      func(*MockTicketRepository)
		expectedStatus int
	}{
		{
			name:     "success",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidID",
			ticketID:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "notFound",
			ticketID: uuid.New().String(),
			setupRepo: func(r *MockTicketRepository) {},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			r := chi.NewRouter()
			r.Patch("/tickets/{id}/accept", h.AcceptTicket)

			req := httptest.NewRequest(http.MethodPatch, "/tickets/"+tt.ticketID+"/accept", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("AcceptTicket() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				// Verify status was updated
				ticket := cache.Get(ticketID)
				if ticket == nil || ticket.Status != kitchenstatus.Statuses.Accepted.Code() {
					t.Error("Ticket status not updated to accepted")
				}
			}
		})
	}
}

func TestHandlerStartTicket(t *testing.T) {
	ticketID := uuid.New()

	tests := []struct {
		name           string
		ticketID       string
		setupRepo      func(*MockTicketRepository)
		expectedStatus int
	}{
		{
			name:     "success",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "accepted"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidID",
			ticketID:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "updateError",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "accepted"})
				r.UpdateFunc = func(ctx context.Context, t *Ticket) error {
					return errors.New("update error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			r := chi.NewRouter()
			r.Patch("/tickets/{id}/start", h.StartTicket)

			req := httptest.NewRequest(http.MethodPatch, "/tickets/"+tt.ticketID+"/start", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("StartTicket() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				ticket := cache.Get(ticketID)
				if ticket == nil || ticket.StartedAt == nil {
					t.Error("StartedAt timestamp not set")
				}
			}
		})
	}
}

func TestHandlerReadyTicket(t *testing.T) {
	ticketID := uuid.New()

	tests := []struct {
		name           string
		ticketID       string
		setupRepo      func(*MockTicketRepository)
		expectedStatus int
	}{
		{
			name:     "success",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidID",
			ticketID:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "notFound",
			ticketID: uuid.New().String(),
			setupRepo: func(r *MockTicketRepository) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "updateError",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})
				r.UpdateFunc = func(ctx context.Context, t *Ticket) error {
					return errors.New("update error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			r := chi.NewRouter()
			r.Patch("/tickets/{id}/ready", h.ReadyTicket)

			req := httptest.NewRequest(http.MethodPatch, "/tickets/"+tt.ticketID+"/ready", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ReadyTicket() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				ticket := cache.Get(ticketID)
				if ticket == nil || ticket.FinishedAt == nil {
					t.Error("FinishedAt timestamp not set")
				}
			}
		})
	}
}

func TestHandlerDeliverTicket(t *testing.T) {
	ticketID := uuid.New()

	tests := []struct {
		name           string
		ticketID       string
		setupRepo      func(*MockTicketRepository)
		expectedStatus int
	}{
		{
			name:     "success",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "ready"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidID",
			ticketID:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "notFound",
			ticketID: uuid.New().String(),
			setupRepo: func(r *MockTicketRepository) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "updateError",
			ticketID: ticketID.String(),
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "ready"})
				r.UpdateFunc = func(ctx context.Context, t *Ticket) error {
					return errors.New("update error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			r := chi.NewRouter()
			r.Patch("/tickets/{id}/deliver", h.DeliverTicket)

			req := httptest.NewRequest(http.MethodPatch, "/tickets/"+tt.ticketID+"/deliver", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("DeliverTicket() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				ticket := cache.Get(ticketID)
				if ticket == nil || ticket.DeliveredAt == nil {
					t.Error("DeliveredAt timestamp not set")
				}
			}
		})
	}
}

func TestHandlerStandbyTicket(t *testing.T) {
	ticketID := uuid.New()

	repo := NewMockTicketRepository()
	repo.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})

	cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
	publisher := NewMockPublisher()

	deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
	h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

	r := chi.NewRouter()
	r.Patch("/tickets/{id}/standby", h.StandbyTicket)

	req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/standby", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("StandbyTicket() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerBlockTicket(t *testing.T) {
	ticketID := uuid.New()
	reasonCodeID := uuid.New()

	tests := []struct {
		name           string
		ticketID       string
		body           interface{}
		setupRepo      func(*MockTicketRepository)
		expectedStatus int
	}{
		{
			name:     "successWithReasonAndNotes",
			ticketID: ticketID.String(),
			body: map[string]string{
				"reason_code_id": reasonCodeID.String(),
				"notes":          "Missing ingredient",
			},
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "successWithEmptyBody",
			ticketID: ticketID.String(),
			body:     nil,
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidID",
			ticketID:       "invalid-uuid",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalidJSON",
			ticketID:       ticketID.String(),
			body:           "invalid json",
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "ticketNotFound",
			ticketID: uuid.New().String(),
			body:     nil,
			setupRepo: func(r *MockTicketRepository) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "updateError",
			ticketID: ticketID.String(),
			body:     nil,
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})
				r.UpdateFunc = func(ctx context.Context, t *Ticket) error {
					return errors.New("update error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			r := chi.NewRouter()
			r.Patch("/tickets/{id}/block", h.BlockTicket)

			var bodyBytes []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					bodyBytes = []byte(str)
				} else {
					bodyBytes, _ = json.Marshal(tt.body)
				}
			}

			req := httptest.NewRequest(http.MethodPatch, "/tickets/"+tt.ticketID+"/block", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("BlockTicket() status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestHandlerRejectTicket(t *testing.T) {
	ticketID := uuid.New()

	repo := NewMockTicketRepository()
	repo.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})

	cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
	publisher := NewMockPublisher()

	deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
	h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

	r := chi.NewRouter()
	r.Patch("/tickets/{id}/reject", h.RejectTicket)

	req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/reject", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("RejectTicket() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerCancelTicket(t *testing.T) {
	ticketID := uuid.New()

	repo := NewMockTicketRepository()
	repo.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})

	cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
	publisher := NewMockPublisher()

	deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
	h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

	r := chi.NewRouter()
	r.Patch("/tickets/{id}/cancel", h.CancelTicket)

	req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CancelTicket() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerUpdateTicketStatus(t *testing.T) {
	ticketID := uuid.New()

	tests := []struct {
		name           string
		ticketID       string
		body           interface{}
		setupRepo      func(*MockTicketRepository)
		expectedStatus int
	}{
		{
			name:     "successToStarted",
			ticketID: ticketID.String(),
			body:     map[string]string{"status": kitchenstatus.Statuses.Started.Code()},
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "successToReady",
			ticketID: ticketID.String(),
			body:     map[string]string{"status": kitchenstatus.Statuses.Ready.Code()},
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalidID",
			ticketID:       "invalid-uuid",
			body:           map[string]string{"status": "started"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "emptyStatus",
			ticketID:       ticketID.String(),
			body:           map[string]string{"status": ""},
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalidJSON",
			ticketID:       ticketID.String(),
			body:           "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "ticketNotFound",
			ticketID: uuid.New().String(),
			body:     map[string]string{"status": "started"},
			setupRepo: func(r *MockTicketRepository) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "cannotModifyDelivered",
			ticketID: ticketID.String(),
			body:     map[string]string{"status": "started"},
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: kitchenstatus.Statuses.Delivered.Code()})
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "cannotModifyCancelled",
			ticketID: ticketID.String(),
			body:     map[string]string{"status": "started"},
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: kitchenstatus.Statuses.Cancelled.Code()})
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "cannotDeliverFromReady",
			ticketID: ticketID.String(),
			body:     map[string]string{"status": kitchenstatus.Statuses.Delivered.Code()},
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: kitchenstatus.Statuses.Ready.Code()})
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "updateError",
			ticketID: ticketID.String(),
			body:     map[string]string{"status": "started"},
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})
				r.UpdateFunc = func(ctx context.Context, t *Ticket) error {
					return errors.New("update error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			r := chi.NewRouter()
			r.Patch("/tickets/{id}/status", h.UpdateTicketStatus)

			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPatch, "/tickets/"+tt.ticketID+"/status", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("UpdateTicketStatus() status = %d, want %d, body = %s", w.Code, tt.expectedStatus, w.Body.String())
			}
		})
	}
}

func TestHandlerUpdateTicketStatusSetsTimestamps(t *testing.T) {
	tests := []struct {
		name           string
		newStatus      string
		checkTimestamp string
	}{
		{
			name:           "setsStartedAt",
			newStatus:      kitchenstatus.Statuses.Started.Code(),
			checkTimestamp: "started_at",
		},
		{
			name:           "setsFinishedAt",
			newStatus:      kitchenstatus.Statuses.Ready.Code(),
			checkTimestamp: "finished_at",
		},
		{
			name:           "setsDeliveredAt",
			newStatus:      kitchenstatus.Statuses.Delivered.Code(),
			checkTimestamp: "delivered_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketID := uuid.New()
			repo := NewMockTicketRepository()
			repo.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})

			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			r := chi.NewRouter()
			r.Patch("/tickets/{id}/status", h.UpdateTicketStatus)

			body, _ := json.Marshal(map[string]string{"status": tt.newStatus})
			req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/status", bytes.NewReader(body))
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("UpdateTicketStatus() status = %d, want %d", w.Code, http.StatusOK)
			}

			// Verify timestamp was set
			ticket := cache.Get(ticketID)
			if ticket == nil {
				t.Fatal("Ticket not found in cache")
			}

			switch tt.checkTimestamp {
			case "started_at":
				if ticket.StartedAt == nil {
					t.Error("StartedAt was not set")
				}
			case "finished_at":
				if ticket.FinishedAt == nil {
					t.Error("FinishedAt was not set")
				}
			case "delivered_at":
				if ticket.DeliveredAt == nil {
					t.Error("DeliveredAt was not set")
				}
			}
		})
	}
}

func TestHandlerReloadCache(t *testing.T) {
	tests := []struct {
		name           string
		setupRepo      func(*MockTicketRepository)
		setupCache     func() *TicketStateCache
		expectedStatus int
	}{
		{
			name: "success",
			setupRepo: func(r *MockTicketRepository) {
				r.AddTicket(&Ticket{ID: uuid.New(), Station: "kitchen", Status: "created"})
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			cache := NewTicketStateCache(nil, repo, apt.NewNoopLogger())
			publisher := NewMockPublisher()

			deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
			h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

			req := httptest.NewRequest(http.MethodPost, "/internal/reload-cache", nil)
			w := httptest.NewRecorder()

			h.ReloadCache(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ReloadCache() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				// ReloadCache uses json.NewEncoder directly, not apt.Respond
				if resp["success"] != true {
					t.Errorf("Response success should be true, got: %s", w.Body.String())
				}
			}
		})
	}
}

func TestHandlerPublishStatusChange(t *testing.T) {
	repo := NewMockTicketRepository()
	cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
	publisher := NewMockPublisher()

	ticketID := uuid.New()
	ticket := &Ticket{
		ID:          ticketID,
		OrderID:     uuid.New(),
		OrderItemID: uuid.New(),
		MenuItemID:  uuid.New(),
		Station:     "kitchen",
		Status:      "started",
	}
	repo.AddTicket(ticket)

	deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
	h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

	r := chi.NewRouter()
	r.Patch("/tickets/{id}/start", h.StartTicket)

	req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/start", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if len(publisher.PublishedEvents) != 1 {
		t.Errorf("Expected 1 published event, got %d", len(publisher.PublishedEvents))
	}
}

func TestHandlerPublishStatusChangeError(t *testing.T) {
	repo := NewMockTicketRepository()
	cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
	publisher := NewMockPublisher()
	publisher.PublishFunc = func(ctx context.Context, topic string, data []byte) error {
		return errors.New("publish error")
	}

	ticketID := uuid.New()
	repo.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})

	deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
	h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

	r := chi.NewRouter()
	r.Patch("/tickets/{id}/start", h.StartTicket)

	req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/start", nil)
	w := httptest.NewRecorder()

	// Should still succeed - publish error is logged but not returned
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("StartTicket() should succeed even with publish error, got status %d", w.Code)
	}
}

func TestHandlerNilCache(t *testing.T) {
	ticketID := uuid.New()
	repo := NewMockTicketRepository()
	repo.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "created"})

	deps := HandlerDeps{Repo: repo, Cache: nil, Publisher: NewMockPublisher()}
	h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

	r := chi.NewRouter()
	r.Patch("/tickets/{id}/start", h.StartTicket)

	req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/start", nil)
	w := httptest.NewRecorder()

	// Should not panic with nil cache
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("StartTicket() with nil cache status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlerBlockTicketWithReasonCode(t *testing.T) {
	ticketID := uuid.New()
	reasonCodeID := uuid.New()

	repo := NewMockTicketRepository()
	repo.AddTicket(&Ticket{ID: ticketID, Station: "kitchen", Status: "started"})

	cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
	publisher := NewMockPublisher()

	deps := HandlerDeps{Repo: repo, Cache: cache, Publisher: publisher}
	h := NewHandler(deps, apt.NewConfig(), apt.NewNoopLogger())

	r := chi.NewRouter()
	r.Patch("/tickets/{id}/block", h.BlockTicket)

	body, _ := json.Marshal(map[string]string{
		"reason_code_id": reasonCodeID.String(),
		"notes":          "Test notes",
	})
	req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/block", bytes.NewReader(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("BlockTicket() status = %d, want %d", w.Code, http.StatusOK)
	}

	ticket := cache.Get(ticketID)
	if ticket == nil {
		t.Fatal("Ticket not found in cache")
	}
	if ticket.ReasonCodeID == nil || *ticket.ReasonCodeID != reasonCodeID {
		t.Error("ReasonCodeID not set correctly")
	}
	if ticket.Notes != "Test notes" {
		t.Errorf("Notes = %q, want %q", ticket.Notes, "Test notes")
	}
}
