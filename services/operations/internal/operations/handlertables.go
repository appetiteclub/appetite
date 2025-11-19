package operations

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/go-chi/chi/v5"
)

type tablesPageState struct {
	Error   string
	Success string
}

type tableViewModel struct {
	ID          string
	Number      string
	Status      string
	StatusLabel string
	GuestCount  int
	AssignedTo  string
	Bill        string
	UpdatedAt   string
}

type tableFormModal struct {
	Title      string
	Action     string
	Mode       string
	Number     string
	GuestCount string
	Status     string
	Statuses   []string
	Error      string
}

// Tables renders the table management view with live data from the table service.
func (h *Handler) Tables(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.Tables")
	defer finish()

	state := tablesPageState{}
	query := r.URL.Query()
	if query.Get("created") == "1" {
		state.Success = "Table created successfully."
	} else if query.Get("updated") == "1" {
		state.Success = "Table updated successfully."
	} else if query.Get("deleted") == "1" {
		state.Success = "Table deleted successfully."
	}

	h.renderTablesPage(w, r, state)
}

// CreateTable handles the command-side POST that proxies table creation to the table service.
func (h *Handler) CreateTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.CreateTable")
	defer finish()

	if !h.requirePermission(w, r, "tables:write") {
		return
	}

	ctx := r.Context()
	log := h.log()

	if err := r.ParseForm(); err != nil {
		log.Error("cannot parse table form", "error", err)
		form := tableFormModal{Title: "Add table", Action: "/add-table", Mode: "create"}
		h.handleFormError(w, r, form, "Could not read the submitted form.")
		return
	}

	number := strings.TrimSpace(r.FormValue("number"))
	guests := strings.TrimSpace(r.FormValue("guest_count"))
	status := strings.TrimSpace(r.FormValue("status"))

	formData := tableFormModal{
		Title:      "Add table",
		Action:     "/add-table",
		Mode:       "create",
		Number:     number,
		GuestCount: guests,
		Status:     status,
		Statuses:   tableStatuses,
	}

	if number == "" {
		h.handleFormError(w, r, formData, "Table number is required.")
		return
	}

	guestCount := 0
	if guests != "" {
		parsed, err := strconv.Atoi(guests)
		if err != nil || parsed < 0 {
			h.handleFormError(w, r, formData, "Guest count must be a positive number.")
			return
		}
		guestCount = parsed
	}

	payload := map[string]interface{}{
		"number": number,
	}
	if guestCount > 0 {
		payload["guest_count"] = guestCount
	}
	if status != "" {
		payload["status"] = status
	}

	if _, err := h.tableClient.Create(ctx, "tables", payload); err != nil {
		log.Error("table service create failed", "error", err)
		h.handleFormError(w, r, formData, "Could not create the table right now. Please try again.")
		return
	}

	aqm.RedirectOrHeader(w, r, "/list-tables?created=1")
}

// UpdateTable applies changes to an existing table through table service.
func (h *Handler) UpdateTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.UpdateTable")
	defer finish()

	if !h.requirePermission(w, r, "tables:write") {
		return
	}

	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if id == "" {
		h.renderTablesPage(w, r, tablesPageState{Error: "Invalid table identifier."})
		return
	}

	if err := r.ParseForm(); err != nil {
		form := tableFormModal{
			Title:      "Edit table",
			Action:     fmt.Sprintf("/update-table/%s", id),
			Mode:       "edit",
			Number:     r.FormValue("number"),
			GuestCount: r.FormValue("guest_count"),
			Status:     r.FormValue("status"),
			Statuses:   tableStatuses,
		}
		h.handleFormError(w, r, form, "Could not read the submitted form.")
		return
	}

	form := tableFormModal{
		Title:      "Edit table",
		Action:     fmt.Sprintf("/update-table/%s", id),
		Mode:       "edit",
		Number:     strings.TrimSpace(r.FormValue("number")),
		GuestCount: strings.TrimSpace(r.FormValue("guest_count")),
		Status:     strings.TrimSpace(r.FormValue("status")),
		Statuses:   tableStatuses,
	}

	if form.Number == "" {
		h.handleFormError(w, r, form, "Table number is required.")
		return
	}

	payload := map[string]interface{}{
		"number": form.Number,
	}
	if form.Status != "" {
		payload["status"] = form.Status
	}
	if form.GuestCount != "" {
		if parsed, err := strconv.Atoi(form.GuestCount); err == nil && parsed >= 0 {
			payload["guest_count"] = parsed
		} else {
			h.handleFormError(w, r, form, "Guest count must be a positive number.")
			return
		}
	}

	h.log().Info("attempting table update", "table_id", id, "payload", payload)

	if _, err := h.tableClient.Update(ctx, "tables", id, payload); err != nil {
		h.log().Error("table service update failed", "error", err, "table_id", id, "payload", payload)
		h.handleFormError(w, r, form, "Could not update the table right now.")
		return
	}

	aqm.RedirectOrHeader(w, r, "/list-tables?updated=1")
}

// DeleteTable removes a table via the table service.
func (h *Handler) DeleteTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.DeleteTable")
	defer finish()

	if !h.requirePermission(w, r, "tables:delete") {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.renderTablesPage(w, r, tablesPageState{Error: "Invalid table identifier."})
		return
	}

	if err := h.tableClient.Delete(r.Context(), "tables", id); err != nil {
		h.log().Error("table service delete failed", "error", err, "table_id", id)
		h.renderTablesPage(w, r, tablesPageState{Error: "Could not delete the table right now."})
		return
	}

	aqm.RedirectOrHeader(w, r, "/list-tables?deleted=1")
}

// NewTableForm serves the create form via HTMX.
func (h *Handler) NewTableForm(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.NewTableForm")
	defer finish()

	if !h.requirePermission(w, r, "tables:write") {
		return
	}

	if !aqm.IsHTMX(r) {
		http.Redirect(w, r, "/list-tables", http.StatusSeeOther)
		return
	}

	form := tableFormModal{
		Title:  "Add table",
		Action: "/add-table",
		Mode:   "create",
	}

	h.renderTableForm(w, form)
}

// EditTableForm serves the edit form via HTMX.
func (h *Handler) EditTableForm(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.EditTableForm")
	defer finish()

	if !h.requirePermission(w, r, "tables:write") {
		return
	}

	if !aqm.IsHTMX(r) {
		http.Redirect(w, r, "/list-tables", http.StatusSeeOther)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Invalid table identifier", http.StatusBadRequest)
		return
	}

	// Prefer data coming from hx-vals to avoid an extra fetch when we already know the values.
	if err := r.ParseForm(); err == nil {
		number := strings.TrimSpace(r.FormValue("number"))
		if number != "" {
			form := tableFormModal{
				Title:      fmt.Sprintf("Edit %s", number),
				Action:     fmt.Sprintf("/update-table/%s", id),
				Mode:       "edit",
				Number:     number,
				GuestCount: strings.TrimSpace(r.FormValue("guest_count")),
				Status:     strings.TrimSpace(r.FormValue("status")),
				Statuses:   tableStatuses,
			}
			h.renderTableForm(w, form)
			return
		}
	}

	table, err := h.fetchTable(r.Context(), id)
	if err != nil {
		h.log().Error("cannot load table", "error", err, "table_id", id)
		http.Error(w, "Could not load table", http.StatusInternalServerError)
		return
	}

	form := tableFormModal{
		Title:      fmt.Sprintf("Edit %s", table.Number),
		Action:     fmt.Sprintf("/update-table/%s", id),
		Mode:       "edit",
		Number:     table.Number,
		GuestCount: strconv.Itoa(table.GuestCount),
		Status:     table.Status,
		Statuses:   tableStatuses,
	}

	h.renderTableForm(w, form)
}

func (h *Handler) renderTablesPage(w http.ResponseWriter, r *http.Request, state tablesPageState) {
	if !h.requirePermission(w, r, "tables:read") {
		return
	}

	tables, err := h.fetchTableViewModels(r.Context())
	if err != nil {
		h.log().Error("unable to load tables", "error", err)
		if state.Error == "" {
			state.Error = "Could not load tables from the service."
		}
	}

	data := map[string]interface{}{
		"Title":    "Tables",
		"Template": "tables",
		"User":     h.getUserFromSession(r),
		"Tables":   tables,
		"Error":    state.Error,
		"Success":  state.Success,
	}

	h.renderTemplate(w, "tables.html", "base.html", data)
}

func (h *Handler) fetchTableViewModels(ctx context.Context) ([]tableViewModel, error) {
	tables, err := h.tableData.ListTables(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]tableViewModel, 0, len(tables))
	for _, resource := range tables {
		assigned := "-"
		if resource.AssignedTo != nil && *resource.AssignedTo != "" {
			assigned = truncateID(*resource.AssignedTo)
		}

		models = append(models, tableViewModel{
			ID:          resource.ID,
			Number:      resource.Number,
			Status:      resource.Status,
			StatusLabel: humanizeStatus(resource.Status),
			GuestCount:  resource.GuestCount,
			AssignedTo:  assigned,
			Bill:        formatBill(resource.CurrentBill),
			UpdatedAt:   relativeTimeSince(resource.UpdatedAt),
		})
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Number < models[j].Number
	})

	return models, nil
}

func (h *Handler) fetchTable(ctx context.Context, id string) (*tableResource, error) {
	return h.tableData.GetTable(ctx, id)
}

func (h *Handler) renderTableForm(w http.ResponseWriter, data tableFormModal) {
	tmpl, err := h.tmplMgr.Get("tables_form.html")
	if err != nil {
		h.log().Error("error loading table form template", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "tables_form.html", data); err != nil {
		h.log().Error("error rendering table form", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// statusLabels maps internal status codes to human-readable labels
var statusLabels = map[string]string{
	"available":      "Available",
	"open":           "Open",
	"reserved":       "Reserved",
	"cleaning":       "Cleaning",
	"out_of_service": "Out of Service",
}

func humanizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "Unknown"
	}

	if label, ok := statusLabels[status]; ok {
		return label
	}

	// Fallback: capitalize first letter only
	return strings.ToUpper(status[:1]) + status[1:]
}

func formatBill(bill *tableBillResource) string {
	if bill == nil {
		return "-"
	}
	return fmt.Sprintf("$%.2f", bill.Total)
}

func relativeTimeSince(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}

	diff := time.Since(ts)
	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return ts.Format("02 Jan 15:04")
}

func truncateID(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8] + "..."
}

var tableStatuses = []string{
	"available",
	"open",
	"reserved",
	"cleaning",
	"out_of_service",
}

func (h *Handler) handleFormError(w http.ResponseWriter, r *http.Request, data tableFormModal, message string) {
	if aqm.IsHTMX(r) {
		data.Error = message
		h.renderTableForm(w, data)
		return
	}

	h.renderTablesPage(w, r, tablesPageState{Error: message})
}
