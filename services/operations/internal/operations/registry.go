package operations

import (
	"context"
	"regexp"
	"strings"
)

// CommandDefinition defines a command with its variations and handler
type CommandDefinition struct {
	Canonical   string
	Variations  []string
	ShortForms  []string
	Pattern     *regexp.Regexp
	Handler     CommandHandler
	Description string
	MinParams   int
	MaxParams   int
}

// CommandHandler processes a matched command
type CommandHandler func(ctx context.Context, params []string) (*CommandResponse, error)

// CommandRegistry holds all available commands
type CommandRegistry struct {
	commands map[string]*CommandDefinition
	parser   *DeterministicParser
}

// NewCommandRegistry creates and initializes the command registry
func NewCommandRegistry(parser *DeterministicParser) *CommandRegistry {
	r := &CommandRegistry{
		commands: make(map[string]*CommandDefinition),
		parser:   parser,
	}
	r.registerAllCommands()
	return r
}

// FindCommand finds a command by matching input against variations
func (r *CommandRegistry) FindCommand(input string) (*CommandDefinition, []string, bool) {
	// Handle special dot notation for authentication
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, ".") {
		dotContent := strings.TrimPrefix(trimmed, ".")
		if dotContent == "" {
			// Just "." means logout (or prompt for PIN if not logged in)
			if cmd, ok := r.commands["exit"]; ok {
				return cmd, []string{}, true
			}
		} else {
			// ".[pin]" means login with PIN
			if cmd, ok := r.commands["login"]; ok {
				return cmd, []string{dotContent}, true
			}
		}
	}

	normalized := normalizeInput(input)
	tokens := tokenize(normalized)

	if len(tokens) == 0 {
		return nil, nil, false
	}

	// Try exact canonical match first
	if cmd, ok := r.commands[tokens[0]]; ok {
		return cmd, tokens[1:], true
	}

	// Try two-word canonical (e.g., "list tables")
	if len(tokens) >= 2 {
		twoWord := tokens[0] + "-" + tokens[1]
		if cmd, ok := r.commands[twoWord]; ok {
			return cmd, tokens[2:], true
		}
	}

	// Try short forms and variations
	for canonical, cmd := range r.commands {
		// Check short forms
		for _, short := range cmd.ShortForms {
			if tokens[0] == short {
				return cmd, tokens[1:], true
			}
		}

		// Check variations
		for _, variation := range cmd.Variations {
			varTokens := tokenize(variation)
			if matchesVariation(tokens, varTokens) {
				remainingTokens := tokens[len(varTokens):]
				return r.commands[canonical], remainingTokens, true
			}
		}
	}

	return nil, nil, false
}

func (r *CommandRegistry) registerAllCommands() {
	// AUTHENTICATION COMMANDS
	r.register("login", &CommandDefinition{
		Canonical:   "login",
		Variations:  []string{"login", "signin"},
		ShortForms:  []string{},
		Handler:     r.parser.handleLogin,
		Description: "Authenticate with PIN",
		MinParams:   0,
		MaxParams:   1,
	})

	r.register("exit", &CommandDefinition{
		Canonical:   "exit",
		Variations:  []string{"exit", "logout", "signout"},
		ShortForms:  []string{},
		Handler:     r.parser.handleLogout,
		Description: "Logout from current session",
		MinParams:   0,
		MaxParams:   0,
	})

	// Help command
	r.register("help", &CommandDefinition{
		Canonical:   "help",
		Variations:  []string{"help", "ayuda", "pomoc", "?"},
		ShortForms:  []string{"h"},
		Pattern:     regexp.MustCompile(`^(help|ayuda|pomoc|\?|h)$`),
		Handler:     r.parser.handleHelp,
		Description: "Show available commands",
		MinParams:   0,
		MaxParams:   0,
	})

	// ORDER QUERIES
	r.register("list-orders", &CommandDefinition{
		Canonical:   "list-orders",
		Variations:  []string{"list orders", "show orders", "get orders", "listar ordenes", "ordenes", "lista zamówień", "zamówienia"},
		ShortForms:  []string{"lo"},
		Handler:     r.parser.handleListOrders,
		Description: "List all orders",
		MinParams:   0,
		MaxParams:   0,
	})

	r.register("list-active-orders", &CommandDefinition{
		Canonical:   "list-active-orders",
		Variations:  []string{"list active orders", "active orders", "ordenes activas", "aktywne zamówienia"},
		ShortForms:  []string{"lao"},
		Handler:     r.parser.handleListActiveOrders,
		Description: "List currently active orders",
		MinParams:   0,
		MaxParams:   0,
	})

	r.register("get-order", &CommandDefinition{
		Canonical:   "get-order",
		Variations:  []string{"get order", "show order", "order", "orden", "zamówienie", "pokaż zamówienie"},
		ShortForms:  []string{"go"},
		Handler:     r.parser.handleGetOrder,
		Description: "Get details for a specific order",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-order-items", &CommandDefinition{
		Canonical:   "get-order-items",
		Variations:  []string{"get order items", "order items", "items", "pozycje zamówienia"},
		ShortForms:  []string{"gi"},
		Handler:     r.parser.handleGetOrderItems,
		Description: "List all items in an order",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-order-status", &CommandDefinition{
		Canonical:   "get-order-status",
		Variations:  []string{"get order status", "order status", "status", "estado", "status zamówienia"},
		ShortForms:  []string{"gs"},
		Handler:     r.parser.handleGetOrderStatus,
		Description: "Get current order state",
		MinParams:   1,
		MaxParams:   1,
	})

	// ORDER COMMANDS
	r.register("open-order", &CommandDefinition{
		Canonical:   "open-order",
		Variations:  []string{"open order", "create order", "new order", "abrir orden", "otwórz zamówienie"},
		ShortForms:  []string{"oo"},
		Handler:     r.parser.handleOpenOrder,
		Description: "Open a new order for a table",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("close-order", &CommandDefinition{
		Canonical:   "close-order",
		Variations:  []string{"close order", "cerrar orden", "zamknij zamówienie"},
		ShortForms:  []string{"co"},
		Handler:     r.parser.handleCloseOrder,
		Description: "Close an active order",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("cancel-order", &CommandDefinition{
		Canonical:   "cancel-order",
		Variations:  []string{"cancel order", "cancelar orden", "anuluj zamówienie"},
		ShortForms:  []string{"xo"},
		Handler:     r.parser.handleCancelOrder,
		Description: "Cancel an active order",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("add-item", &CommandDefinition{
		Canonical:   "add-item",
		Variations:  []string{"add item", "agregar item", "dodaj pozycję"},
		ShortForms:  []string{"ai"},
		Handler:     r.parser.handleAddItem,
		Description: "Add an item to the order",
		MinParams:   3, // order_id, item_code, quantity
		MaxParams:   3,
	})

	r.register("remove-item", &CommandDefinition{
		Canonical:   "remove-item",
		Variations:  []string{"remove item", "delete item", "eliminar item", "usuń pozycję"},
		ShortForms:  []string{"ri"},
		Handler:     r.parser.handleRemoveItem,
		Description: "Remove an item from the order",
		MinParams:   2, // order_id, item_code
		MaxParams:   2,
	})

	r.register("update-item", &CommandDefinition{
		Canonical:   "update-item",
		Variations:  []string{"update item", "actualizar item", "zaktualizuj pozycję"},
		ShortForms:  []string{"ui"},
		Handler:     r.parser.handleUpdateItem,
		Description: "Update quantity for an item",
		MinParams:   3, // order_id, item_code, quantity
		MaxParams:   3,
	})

	r.register("send-to-kitchen", &CommandDefinition{
		Canonical:   "send-to-kitchen",
		Variations:  []string{"send to kitchen", "send kitchen", "enviar cocina", "wyślij do kuchni"},
		ShortForms:  []string{"sk"},
		Handler:     r.parser.handleSendToKitchen,
		Description: "Send order to kitchen",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("mark-ready", &CommandDefinition{
		Canonical:   "mark-ready",
		Variations:  []string{"mark ready", "ready", "listo", "gotowe"},
		ShortForms:  []string{"mr"},
		Handler:     r.parser.handleMarkReady,
		Description: "Mark order as ready for delivery",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("reopen-order", &CommandDefinition{
		Canonical:   "reopen-order",
		Variations:  []string{"reopen order", "reabrir orden", "ponownie otwórz zamówienie"},
		ShortForms:  []string{"ro"},
		Handler:     r.parser.handleReopenOrder,
		Description: "Reopen a closed order",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("add-note", &CommandDefinition{
		Canonical:   "add-note",
		Variations:  []string{"add note", "agregar nota", "dodaj notatkę"},
		ShortForms:  []string{"an"},
		Handler:     r.parser.handleAddNote,
		Description: "Add a note to an order",
		MinParams:   2, // order_id, note
		MaxParams:   2,
	})

	r.register("assign-order", &CommandDefinition{
		Canonical:   "assign-order",
		Variations:  []string{"assign order", "asignar orden", "przypisz zamówienie"},
		ShortForms:  []string{"ao"},
		Handler:     r.parser.handleAssignOrder,
		Description: "Assign waiter to an order",
		MinParams:   2, // order_id, user_id
		MaxParams:   2,
	})

	r.register("split-order", &CommandDefinition{
		Canonical:   "split-order",
		Variations:  []string{"split order", "dividir orden", "podziel zamówienie"},
		ShortForms:  []string{"so"},
		Handler:     r.parser.handleSplitOrder,
		Description: "Split an order by strategy",
		MinParams:   2, // order_id, strategy
		MaxParams:   2,
	})

	r.register("merge-orders", &CommandDefinition{
		Canonical:   "merge-orders",
		Variations:  []string{"merge orders", "combinar ordenes", "połącz zamówienia"},
		ShortForms:  []string{"mo"},
		Handler:     r.parser.handleMergeOrders,
		Description: "Merge multiple orders into one",
		MinParams:   2, // order_id_1, order_id_2
		MaxParams:   2,
	})

	r.register("create-group", &CommandDefinition{
		Canonical:   "create-group",
		Variations:  []string{"create group", "crear grupo", "utwórz grupę"},
		ShortForms:  []string{"cg"},
		Handler:     r.parser.handleCreateGroup,
		Description: "Create a subgroup for partial billing",
		MinParams:   2, // order_id, group_label
		MaxParams:   2,
	})

	r.register("add-item-to-group", &CommandDefinition{
		Canonical:   "add-item-to-group",
		Variations:  []string{"add item to group", "agregar item a grupo", "dodaj pozycję do grupy"},
		ShortForms:  []string{"aig"},
		Handler:     r.parser.handleAddItemToGroup,
		Description: "Add an item to a specific group",
		MinParams:   4, // order_id, group_label, item_code, quantity
		MaxParams:   4,
	})

	r.register("move-item-to-group", &CommandDefinition{
		Canonical:   "move-item-to-group",
		Variations:  []string{"move item to group", "mover item a grupo", "przenieś pozycję do grupy"},
		ShortForms:  []string{"mig"},
		Handler:     r.parser.handleMoveItemToGroup,
		Description: "Move an item to another group",
		MinParams:   3, // order_id, item_code, target_group
		MaxParams:   3,
	})

	r.register("remove-group", &CommandDefinition{
		Canonical:   "remove-group",
		Variations:  []string{"remove group", "eliminar grupo", "usuń grupę"},
		ShortForms:  []string{"rg"},
		Handler:     r.parser.handleRemoveGroup,
		Description: "Remove an existing group",
		MinParams:   2, // order_id, group_label
		MaxParams:   2,
	})

	r.register("apply-discount", &CommandDefinition{
		Canonical:   "apply-discount",
		Variations:  []string{"apply discount", "aplicar descuento", "zastosuj zniżkę"},
		ShortForms:  []string{"ad"},
		Handler:     r.parser.handleApplyDiscount,
		Description: "Apply a discount to an order",
		MinParams:   2, // order_id, amount|percent
		MaxParams:   2,
	})

	r.register("transfer-order", &CommandDefinition{
		Canonical:   "transfer-order",
		Variations:  []string{"transfer order", "transferir orden", "przenieś zamówienie"},
		ShortForms:  []string{"to"},
		Handler:     r.parser.handleTransferOrder,
		Description: "Move an order to another table",
		MinParams:   2, // order_id, new_table_id
		MaxParams:   2,
	})

	// ADDITIONAL ORDER QUERIES

	r.register("get-orders-by-table", &CommandDefinition{
		Canonical:   "get-orders-by-table",
		Variations:  []string{"get orders by table", "orders by table", "ordenes por mesa", "zamówienia stolika"},
		ShortForms:  []string{"got"},
		Handler:     r.parser.handleGetOrdersByTable,
		Description: "Get all orders for a specific table",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-groups", &CommandDefinition{
		Canonical:   "get-groups",
		Variations:  []string{"get groups", "grupos", "grupy"},
		ShortForms:  []string{"gg"},
		Handler:     r.parser.handleGetGroups,
		Description: "Get groups within an order",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-order-history", &CommandDefinition{
		Canonical:   "get-order-history",
		Variations:  []string{"get order history", "order history", "historial orden", "historia zamówienia"},
		ShortForms:  []string{"gh"},
		Handler:     r.parser.handleGetOrderHistory,
		Description: "Retrieve order lifecycle history",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-waiter", &CommandDefinition{
		Canonical:   "get-waiter",
		Variations:  []string{"get waiter", "mesero", "kelner"},
		ShortForms:  []string{"gw"},
		Handler:     r.parser.handleGetWaiter,
		Description: "Get assigned waiter for order",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-order-notes", &CommandDefinition{
		Canonical:   "get-order-notes",
		Variations:  []string{"get order notes", "order notes", "notas orden", "notatki zamówienia"},
		ShortForms:  []string{"gn"},
		Handler:     r.parser.handleGetOrderNotes,
		Description: "Retrieve all notes for an order",
		MinParams:   1,
		MaxParams:   1,
	})

	// TABLE QUERIES
	r.register("list-tables", &CommandDefinition{
		Canonical:   "list-tables",
		Variations:  []string{"list tables", "show tables", "get tables", "tables", "mesas", "listar mesas", "stoliki", "lista stolików"},
		ShortForms:  []string{"lt"},
		Handler:     r.parser.handleListTables,
		Description: "List all tables",
		MinParams:   0,
		MaxParams:   0,
	})

	r.register("list-available-tables", &CommandDefinition{
		Canonical:   "list-available-tables",
		Variations:  []string{"list available tables", "available tables", "mesas disponibles", "dostępne stoliki"},
		ShortForms:  []string{"lat"},
		Handler:     r.parser.handleListAvailableTables,
		Description: "List currently available tables",
		MinParams:   0,
		MaxParams:   0,
	})

	r.register("list-occupied-tables", &CommandDefinition{
		Canonical:   "list-occupied-tables",
		Variations:  []string{"list occupied tables", "occupied tables", "mesas ocupadas", "zajęte stoliki"},
		ShortForms:  []string{"lot"},
		Handler:     r.parser.handleListOccupiedTables,
		Description: "List tables currently in use",
		MinParams:   0,
		MaxParams:   0,
	})

	r.register("get-table", &CommandDefinition{
		Canonical:   "get-table",
		Variations:  []string{"get table", "show table", "table", "mesa", "stolik", "pokaż stolik"},
		ShortForms:  []string{"gt"},
		Handler:     r.parser.handleGetTable,
		Description: "Get detailed info for a specific table",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-table-status", &CommandDefinition{
		Canonical:   "get-table-status",
		Variations:  []string{"get table status", "table status", "status stolika"},
		ShortForms:  []string{"gts"},
		Handler:     r.parser.handleGetTableStatus,
		Description: "Fetch current table status",
		MinParams:   1,
		MaxParams:   1,
	})

	// TABLE COMMANDS
	r.register("seat-party", &CommandDefinition{
		Canonical:   "seat-party",
		Variations:  []string{"seat party", "seat", "sentar", "posadź gości"},
		ShortForms:  []string{"sp"},
		Handler:     r.parser.handleSeatParty,
		Description: "Seat a party at the table",
		MinParams:   2, // table_id, party_size
		MaxParams:   2,
	})

	r.register("release-table", &CommandDefinition{
		Canonical:   "release-table",
		Variations:  []string{"release table", "free table", "liberar mesa", "zwolnij stolik"},
		ShortForms:  []string{"rt"},
		Handler:     r.parser.handleReleaseTable,
		Description: "Release a table once it's free",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("reserve-table", &CommandDefinition{
		Canonical:   "reserve-table",
		Variations:  []string{"reserve table", "book table", "reservar mesa", "zarezerwuj stolik"},
		ShortForms:  []string{"rv"},
		Handler:     r.parser.handleReserveTable,
		Description: "Reserve a table for a customer",
		MinParams:   1, // table_id (customer_name optional)
		MaxParams:   2,
	})

	r.register("cancel-reservation", &CommandDefinition{
		Canonical:   "cancel-reservation",
		Variations:  []string{"cancel reservation", "cancelar reserva", "anuluj rezerwację"},
		ShortForms:  []string{"cr"},
		Handler:     r.parser.handleCancelReservation,
		Description: "Cancel a table reservation",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("assign-waiter", &CommandDefinition{
		Canonical:   "assign-waiter",
		Variations:  []string{"assign waiter", "asignar mesero", "przypisz kelnera"},
		ShortForms:  []string{"aw"},
		Handler:     r.parser.handleAssignWaiter,
		Description: "Assign waiter to a table",
		MinParams:   2, // table_id, user_id
		MaxParams:   2,
	})

	r.register("mark-table-clean", &CommandDefinition{
		Canonical:   "mark-table-clean",
		Variations:  []string{"mark table clean", "clean table", "mesa limpia", "stolik czysty"},
		ShortForms:  []string{"mtc"},
		Handler:     r.parser.handleMarkTableClean,
		Description: "Mark table as cleaned and ready",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("mark-table-dirty", &CommandDefinition{
		Canonical:   "mark-table-dirty",
		Variations:  []string{"mark table dirty", "dirty table", "mesa sucia", "stół brudny"},
		ShortForms:  []string{"mtd"},
		Handler:     r.parser.handleMarkTableDirty,
		Description: "Mark table as dirty after use",
		MinParams:   1,
		MaxParams:   1,
	})

	// ADDITIONAL TABLE QUERIES
	r.register("get-table-orders", &CommandDefinition{
		Canonical:   "get-table-orders",
		Variations:  []string{"get table orders", "table orders", "ordenes mesa", "zamówienia stolika"},
		ShortForms:  []string{"gto"},
		Handler:     r.parser.handleGetTableOrders,
		Description: "Get all orders for a table",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-table-history", &CommandDefinition{
		Canonical:   "get-table-history",
		Variations:  []string{"get table history", "table history", "historial mesa", "historia stolika"},
		ShortForms:  []string{"gth"},
		Handler:     r.parser.handleGetTableHistory,
		Description: "Get table activity history",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-table-server", &CommandDefinition{
		Canonical:   "get-table-server",
		Variations:  []string{"get table server", "table server", "mesero mesa", "kelner stolika"},
		ShortForms:  []string{"gts"},
		Handler:     r.parser.handleGetTableServer,
		Description: "Get assigned server for table",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("get-reservations", &CommandDefinition{
		Canonical:   "get-reservations",
		Variations:  []string{"get reservations", "reservations", "reservas", "rezerwacje"},
		ShortForms:  []string{"gr"},
		Handler:     r.parser.handleGetReservations,
		Description: "Get all current reservations",
		MinParams:   0,
		MaxParams:   0,
	})

	r.register("get-table-capacity", &CommandDefinition{
		Canonical:   "get-table-capacity",
		Variations:  []string{"get table capacity", "table capacity", "capacidad mesa", "pojemność stolika"},
		ShortForms:  []string{"gtc"},
		Handler:     r.parser.handleGetTableCapacity,
		Description: "Get table seating capacity",
		MinParams:   1,
		MaxParams:   1,
	})

	// ADDITIONAL TABLE COMMANDS
	r.register("create-table", &CommandDefinition{
		Canonical:   "create-table",
		Variations:  []string{"create table", "new table", "crear mesa", "utwórz stolik"},
		ShortForms:  []string{"ct"},
		Handler:     r.parser.handleCreateTable,
		Description: "Create a new table",
		MinParams:   2, // table_id, capacity
		MaxParams:   2,
	})

	r.register("delete-table", &CommandDefinition{
		Canonical:   "delete-table",
		Variations:  []string{"delete table", "remove table", "eliminar mesa", "usuń stolik"},
		ShortForms:  []string{"dt"},
		Handler:     r.parser.handleDeleteTable,
		Description: "Delete a table",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("update-table-capacity", &CommandDefinition{
		Canonical:   "update-table-capacity",
		Variations:  []string{"update table capacity", "change capacity", "actualizar capacidad", "zmień pojemność"},
		ShortForms:  []string{"utc"},
		Handler:     r.parser.handleUpdateTableCapacity,
		Description: "Update table seating capacity",
		MinParams:   2, // table_id, new_capacity
		MaxParams:   2,
	})

	r.register("rename-table", &CommandDefinition{
		Canonical:   "rename-table",
		Variations:  []string{"rename table", "cambiar nombre mesa", "zmień nazwę stolika"},
		ShortForms:  []string{"rnt"},
		Handler:     r.parser.handleRenameTable,
		Description: "Rename a table",
		MinParams:   2, // table_id, new_name
		MaxParams:   2,
	})

	r.register("set-table-location", &CommandDefinition{
		Canonical:   "set-table-location",
		Variations:  []string{"set table location", "ubicar mesa", "ustaw lokalizację stolika"},
		ShortForms:  []string{"stl"},
		Handler:     r.parser.handleSetTableLocation,
		Description: "Set table location/zone",
		MinParams:   2, // table_id, location
		MaxParams:   2,
	})

	r.register("merge-tables", &CommandDefinition{
		Canonical:   "merge-tables",
		Variations:  []string{"merge tables", "combinar mesas", "połącz stoliki"},
		ShortForms:  []string{"mt"},
		Handler:     r.parser.handleMergeTables,
		Description: "Merge multiple tables",
		MinParams:   2, // table_id_1, table_id_2
		MaxParams:   2,
	})

	r.register("unmerge-tables", &CommandDefinition{
		Canonical:   "unmerge-tables",
		Variations:  []string{"unmerge tables", "separar mesas", "rozdziel stoliki"},
		ShortForms:  []string{"umt"},
		Handler:     r.parser.handleUnmergeTables,
		Description: "Unmerge previously merged tables",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("block-table", &CommandDefinition{
		Canonical:   "block-table",
		Variations:  []string{"block table", "bloquear mesa", "zablokuj stolik"},
		ShortForms:  []string{"bt"},
		Handler:     r.parser.handleBlockTable,
		Description: "Block a table from use",
		MinParams:   1, // table_id (reason optional)
		MaxParams:   2,
	})

	r.register("unblock-table", &CommandDefinition{
		Canonical:   "unblock-table",
		Variations:  []string{"unblock table", "desbloquear mesa", "odblokuj stolik"},
		ShortForms:  []string{"ubt"},
		Handler:     r.parser.handleUnblockTable,
		Description: "Unblock a previously blocked table",
		MinParams:   1,
		MaxParams:   1,
	})

	r.register("transfer-table", &CommandDefinition{
		Canonical:   "transfer-table",
		Variations:  []string{"transfer table", "transferir mesa", "przenieś stolik"},
		ShortForms:  []string{"tt"},
		Handler:     r.parser.handleTransferTable,
		Description: "Transfer table to another waiter",
		MinParams:   2, // table_id, new_waiter_id
		MaxParams:   2,
	})

	r.register("set-table-note", &CommandDefinition{
		Canonical:   "set-table-note",
		Variations:  []string{"set table note", "nota mesa", "notatka stolika"},
		ShortForms:  []string{"stn"},
		Handler:     r.parser.handleSetTableNote,
		Description: "Add a note to a table",
		MinParams:   2, // table_id, note
		MaxParams:   2,
	})

	// UTILITY COMMANDS

	r.register("undo", &CommandDefinition{
		Canonical:   "undo",
		Variations:  []string{"undo", "deshacer", "cofnij"},
		ShortForms:  []string{"u", "un"},
		Handler:     r.parser.handleUndo,
		Description: "Undo the last command",
		MinParams:   0,
		MaxParams:   0,
	})
}

func (r *CommandRegistry) register(canonical string, def *CommandDefinition) {
	r.commands[canonical] = def
}

// Helper functions
func normalizeInput(input string) string {
	// Lowercase
	s := strings.ToLower(input)
	// Trim spaces
	s = strings.TrimSpace(s)
	// Replace multiple spaces with single space
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	// Replace hyphens with spaces for matching
	s = strings.ReplaceAll(s, "-", " ")
	return s
}

func tokenize(input string) []string {
	tokens := strings.Fields(input)
	return tokens
}

func matchesVariation(inputTokens, variationTokens []string) bool {
	if len(inputTokens) < len(variationTokens) {
		return false
	}
	for i, vt := range variationTokens {
		if inputTokens[i] != vt {
			return false
		}
	}
	return true
}
