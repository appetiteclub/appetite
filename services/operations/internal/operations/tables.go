package operations

import (
	"context"
	"fmt"
)

// TABLE QUERIES

func (p *DeterministicParser) handleListTables(ctx context.Context, params []string) (*CommandResponse, error) {
	// TODO: Call table service GET /tables

	html := `
		<p><strong>All Tables:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Table #</th>
					<th>Capacity</th>
					<th>Status</th>
					<th>Party Size</th>
					<th>Server</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Table 1</td>
					<td>4</td>
					<td><span style="color: #10b981">Available</span></td>
					<td>-</td>
					<td>-</td>
				</tr>
				<tr>
					<td>Table 2</td>
					<td>6</td>
					<td><span style="color: #f59e0b">Occupied</span></td>
					<td>5</td>
					<td>Maria</td>
				</tr>
				<tr>
					<td>Table 3</td>
					<td>2</td>
					<td><span style="color: #ef4444">Reserved</span></td>
					<td>2</td>
					<td>John</td>
				</tr>
			</tbody>
		</table>
		<p><em>Total: 3 tables (1 available, 1 occupied, 1 reserved)</em></p>
	`

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Tables retrieved successfully",
	}, nil
}

func (p *DeterministicParser) handleListAvailableTables(ctx context.Context, params []string) (*CommandResponse, error) {
	// TODO: Call table service GET /tables?status=available

	html := `
		<p><strong>Available Tables:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Table #</th>
					<th>Capacity</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Table 1</td>
					<td>4</td>
					<td><span style="color: #10b981">Available</span></td>
				</tr>
				<tr>
					<td>Table 5</td>
					<td>8</td>
					<td><span style="color: #10b981">Available</span></td>
				</tr>
			</tbody>
		</table>
		<p><em>2 tables available</em></p>
	`

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Available tables retrieved",
	}, nil
}

func (p *DeterministicParser) handleListOccupiedTables(ctx context.Context, params []string) (*CommandResponse, error) {
	// TODO: Call table service GET /tables?status=occupied

	html := `
		<p><strong>Occupied Tables:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Table #</th>
					<th>Capacity</th>
					<th>Party Size</th>
					<th>Server</th>
					<th>Duration</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Table 2</td>
					<td>6</td>
					<td>5</td>
					<td>Maria</td>
					<td>45 min</td>
				</tr>
				<tr>
					<td>Table 4</td>
					<td>4</td>
					<td>3</td>
					<td>John</td>
					<td>20 min</td>
				</tr>
			</tbody>
		</table>
		<p><em>2 tables occupied</em></p>
	`

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Occupied tables retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service GET /tables/:id

	html := fmt.Sprintf(`
		<p><strong>Table %s Details:</strong></p>
		<ul>
			<li><strong>Capacity:</strong> 6 people</li>
			<li><strong>Status:</strong> <span style="color: #f59e0b">Occupied</span></li>
			<li><strong>Party Size:</strong> 5 people</li>
			<li><strong>Server:</strong> Maria Rodriguez</li>
			<li><strong>Seated At:</strong> 45 minutes ago</li>
			<li><strong>Active Orders:</strong> 1 (ORD-101)</li>
			<li><strong>Total Bill:</strong> $78.50</li>
		</ul>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s details retrieved", tableID),
	}, nil
}

func (p *DeterministicParser) handleGetTableStatus(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service GET /tables/:id/status

	html := fmt.Sprintf(`
		<p><strong>Table %s Status:</strong></p>
		<ul>
			<li><strong>Current State:</strong> <span style="color: #f59e0b">Occupied</span></li>
			<li><strong>Clean Status:</strong> Clean</li>
			<li><strong>Party Size:</strong> 5/6</li>
			<li><strong>Duration:</strong> 45 minutes</li>
			<li><strong>Server Assigned:</strong> Yes (Maria)</li>
		</ul>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Table status retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetTableOrders(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service GET /tables/:id/orders

	html := fmt.Sprintf(`
		<p><strong>Orders for Table %s:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Order #</th>
					<th>Status</th>
					<th>Items</th>
					<th>Total</th>
					<th>Created</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>ORD-101</td>
					<td><span style="color: #f59e0b">Preparing</span></td>
					<td>3</td>
					<td>$45.00</td>
					<td>10 min ago</td>
				</tr>
			</tbody>
		</table>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Table orders retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetTableHistory(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service GET /tables/:id/history

	html := fmt.Sprintf(`
		<p><strong>Table %s History (Today):</strong></p>
		<ul>
			<li><strong>9:00 AM - 10:30 AM:</strong> Party of 2, Server: John, Total: $35.00</li>
			<li><strong>11:00 AM - 12:15 PM:</strong> Party of 4, Server: Maria, Total: $68.50</li>
			<li><strong>1:00 PM - Current:</strong> Party of 5, Server: Maria, Total: $78.50</li>
		</ul>
		<p><em>3 seatings today, $182.00 total revenue</em></p>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Table history retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetTableServer(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service GET /tables/:id/server

	html := fmt.Sprintf(`
		<p><strong>Server for Table %s:</strong></p>
		<ul>
			<li><strong>Name:</strong> Maria Rodriguez</li>
			<li><strong>User ID:</strong> USR-123</li>
			<li><strong>Assigned:</strong> 45 minutes ago</li>
			<li><strong>Current Tables:</strong> 3 (Table 2, Table 6, Table 8)</li>
		</ul>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Table server info retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetReservations(ctx context.Context, params []string) (*CommandResponse, error) {
	// TODO: Call table service GET /tables/reservations

	html := `
		<p><strong>Current Reservations:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Table #</th>
					<th>Customer</th>
					<th>Party Size</th>
					<th>Time</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Table 3</td>
					<td>Smith Family</td>
					<td>2</td>
					<td>6:00 PM</td>
					<td><span style="color: #ef4444">Reserved</span></td>
				</tr>
				<tr>
					<td>Table 7</td>
					<td>Johnson Party</td>
					<td>6</td>
					<td>7:30 PM</td>
					<td><span style="color: #ef4444">Reserved</span></td>
				</tr>
			</tbody>
		</table>
		<p><em>2 reservations</em></p>
	`

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Reservations retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetTableCapacity(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service GET /tables/:id/capacity

	html := fmt.Sprintf(`
		<p><strong>Table %s Capacity:</strong></p>
		<ul>
			<li><strong>Standard Capacity:</strong> 6 people</li>
			<li><strong>Maximum (with extra chairs):</strong> 8 people</li>
			<li><strong>Minimum:</strong> 4 people</li>
			<li><strong>Configuration:</strong> Square table</li>
		</ul>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Table capacity info retrieved",
	}, nil
}

// TABLE COMMANDS

func (p *DeterministicParser) handleSeatParty(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	partySize := params[1]
	// TODO: Call table service POST /tables/:id/seat

	html := fmt.Sprintf(`
		<p>✅ <strong>Party Seated Successfully</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Party Size:</strong> %s people</li>
			<li><strong>Status:</strong> <span style="color: #f59e0b">Occupied</span></li>
			<li><strong>Seated At:</strong> Just now</li>
		</ul>
		<p><em>Use <code>open-order %s</code> to start an order</em></p>
	`, tableID, partySize, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Party of %s seated at table %s", partySize, tableID),
	}, nil
}

func (p *DeterministicParser) handleReleaseTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service POST /tables/:id/release

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Released Successfully</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Status:</strong> <span style="color: #10b981">Available</span></li>
			<li><strong>Duration:</strong> 1h 15m</li>
			<li><strong>Released At:</strong> Just now</li>
		</ul>
		<p><em>Table ready for cleaning and next party</em></p>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s released", tableID),
	}, nil
}

func (p *DeterministicParser) handleReserveTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	customerName := ""
	if len(params) > 1 {
		customerName = params[1]
	}
	// TODO: Call table service POST /tables/:id/reserve

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Reserved Successfully</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Customer:</strong> %s</li>
			<li><strong>Status:</strong> <span style="color: #ef4444">Reserved</span></li>
			<li><strong>Reserved At:</strong> Just now</li>
		</ul>
	`, tableID, customerName)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s reserved", tableID),
	}, nil
}

func (p *DeterministicParser) handleCancelReservation(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service POST /tables/:id/cancel-reservation

	html := fmt.Sprintf(`
		<p>⚠️ <strong>Reservation Cancelled</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Status:</strong> <span style="color: #10b981">Available</span></li>
			<li><strong>Cancelled At:</strong> Just now</li>
		</ul>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Reservation for table %s cancelled", tableID),
	}, nil
}

func (p *DeterministicParser) handleAssignWaiter(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	userID := params[1]
	// TODO: Call table service POST /tables/:id/assign

	html := fmt.Sprintf(`
		<p>✅ <strong>Waiter Assigned</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Waiter:</strong> User %s</li>
			<li><strong>Assigned At:</strong> Just now</li>
		</ul>
	`, tableID, userID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Waiter assigned successfully",
	}, nil
}

func (p *DeterministicParser) handleMarkTableClean(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service POST /tables/:id/clean

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Marked as Clean</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Status:</strong> Clean and ready</li>
			<li><strong>Marked At:</strong> Just now</li>
		</ul>
		<p><em>Table is ready for next party</em></p>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s marked as clean", tableID),
	}, nil
}

func (p *DeterministicParser) handleMarkTableDirty(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service POST /tables/:id/dirty

	html := fmt.Sprintf(`
		<p>⚠️ <strong>Table Marked as Dirty</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Status:</strong> Needs cleaning</li>
			<li><strong>Marked At:</strong> Just now</li>
		</ul>
		<p><em>Housekeeping has been notified</em></p>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s marked as dirty", tableID),
	}, nil
}

// ADDITIONAL TABLE COMMANDS FROM SPEC

func (p *DeterministicParser) handleCreateTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	capacity := params[1]
	// TODO: Call table service POST /tables

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Created Successfully</strong></p>
		<ul>
			<li><strong>Table ID:</strong> Table %s</li>
			<li><strong>Capacity:</strong> %s people</li>
			<li><strong>Status:</strong> <span style="color: #10b981">Available</span></li>
			<li><strong>Created At:</strong> Just now</li>
		</ul>
	`, tableID, capacity)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s created", tableID),
	}, nil
}

func (p *DeterministicParser) handleDeleteTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service DELETE /tables/:id

	html := fmt.Sprintf(`
		<p>⚠️ <strong>Table Deleted</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Status:</strong> Removed from system</li>
		</ul>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s deleted", tableID),
	}, nil
}

func (p *DeterministicParser) handleUpdateTableCapacity(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	newCapacity := params[1]
	// TODO: Call table service PATCH /tables/:id/capacity

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Capacity Updated</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>New Capacity:</strong> %s people</li>
			<li><strong>Updated At:</strong> Just now</li>
		</ul>
	`, tableID, newCapacity)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Table capacity updated",
	}, nil
}

func (p *DeterministicParser) handleRenameTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	newName := params[1]
	// TODO: Call table service PATCH /tables/:id/name

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Renamed</strong></p>
		<ul>
			<li><strong>Old ID:</strong> Table %s</li>
			<li><strong>New Name:</strong> %s</li>
			<li><strong>Updated At:</strong> Just now</li>
		</ul>
	`, tableID, newName)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Table renamed successfully",
	}, nil
}

func (p *DeterministicParser) handleSetTableLocation(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	location := params[1]
	// TODO: Call table service PATCH /tables/:id/location

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Location Updated</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Location:</strong> %s</li>
			<li><strong>Updated At:</strong> Just now</li>
		</ul>
	`, tableID, location)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Table location updated",
	}, nil
}

func (p *DeterministicParser) handleMergeTables(ctx context.Context, params []string) (*CommandResponse, error) {
	table1 := params[0]
	table2 := params[1]
	// TODO: Call table service POST /tables/merge

	html := fmt.Sprintf(`
		<p>✅ <strong>Tables Merged</strong></p>
		<ul>
			<li><strong>Tables:</strong> Table %s + Table %s</li>
			<li><strong>New Combined ID:</strong> Table %s-%s</li>
			<li><strong>Combined Capacity:</strong> Calculating...</li>
		</ul>
	`, table1, table2, table1, table2)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Tables merged successfully",
	}, nil
}

func (p *DeterministicParser) handleUnmergeTables(ctx context.Context, params []string) (*CommandResponse, error) {
	mergedTableID := params[0]
	// TODO: Call table service POST /tables/:id/unmerge

	html := fmt.Sprintf(`
		<p>✅ <strong>Tables Unmerged</strong></p>
		<ul>
			<li><strong>Merged Table:</strong> Table %s</li>
			<li><strong>Status:</strong> Restored to individual tables</li>
		</ul>
	`, mergedTableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Tables unmerged successfully",
	}, nil
}

func (p *DeterministicParser) handleBlockTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	reason := ""
	if len(params) > 1 {
		reason = params[1]
	}
	// TODO: Call table service POST /tables/:id/block

	html := fmt.Sprintf(`
		<p>⚠️ <strong>Table Blocked</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Reason:</strong> %s</li>
			<li><strong>Status:</strong> <span style="color: #ef4444">Blocked</span></li>
		</ul>
		<p><em>Table cannot be seated until unblocked</em></p>
	`, tableID, reason)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s blocked", tableID),
	}, nil
}

func (p *DeterministicParser) handleUnblockTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call table service POST /tables/:id/unblock

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Unblocked</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Status:</strong> <span style="color: #10b981">Available</span></li>
		</ul>
		<p><em>Table can now be seated</em></p>
	`, tableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s unblocked", tableID),
	}, nil
}

func (p *DeterministicParser) handleTransferTable(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	newWaiterID := params[1]
	// TODO: Call table service POST /tables/:id/transfer

	html := fmt.Sprintf(`
		<p>✅ <strong>Table Transferred</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>New Waiter:</strong> User %s</li>
			<li><strong>Transferred At:</strong> Just now</li>
		</ul>
	`, tableID, newWaiterID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Table %s transferred", tableID),
	}, nil
}

func (p *DeterministicParser) handleSetTableNote(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	note := params[1]
	// TODO: Call table service POST /tables/:id/note

	html := fmt.Sprintf(`
		<p>✅ <strong>Note Added to Table</strong></p>
		<ul>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Note:</strong> %s</li>
			<li><strong>Added At:</strong> Just now</li>
		</ul>
	`, tableID, note)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Note added to table",
	}, nil
}
