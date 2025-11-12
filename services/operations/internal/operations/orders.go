package operations

import (
	"context"
	"fmt"
)

// ORDER QUERIES

func (p *DeterministicParser) handleListOrders(ctx context.Context, params []string) (*CommandResponse, error) {
	// TODO: Call order service GET /orders
	html := `
		<p><strong>All Orders:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Order #</th>
					<th>Table</th>
					<th>Items</th>
					<th>Status</th>
					<th>Total</th>
					<th>Created</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>101</td>
					<td>Table 2</td>
					<td>3</td>
					<td><span style="color: #f59e0b">Preparing</span></td>
					<td>$45.00</td>
					<td>10 min ago</td>
				</tr>
				<tr>
					<td>102</td>
					<td>Table 3</td>
					<td>5</td>
					<td><span style="color: #3b82f6">Pending</span></td>
					<td>$78.50</td>
					<td>5 min ago</td>
				</tr>
				<tr>
					<td>103</td>
					<td>Table 5</td>
					<td>2</td>
					<td><span style="color: #10b981">Ready</span></td>
					<td>$32.00</td>
					<td>2 min ago</td>
				</tr>
			</tbody>
		</table>
		<p><em>Total: 3 orders</em></p>
	`

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Orders retrieved successfully",
	}, nil
}

func (p *DeterministicParser) handleListActiveOrders(ctx context.Context, params []string) (*CommandResponse, error) {
	// TODO: Call order service GET /orders/active
	html := `
		<p><strong>Active Orders:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Order #</th>
					<th>Table</th>
					<th>Items</th>
					<th>Status</th>
					<th>Total</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>101</td>
					<td>Table 2</td>
					<td>3</td>
					<td><span style="color: #f59e0b">Preparing</span></td>
					<td>$45.00</td>
				</tr>
				<tr>
					<td>102</td>
					<td>Table 3</td>
					<td>5</td>
					<td><span style="color: #3b82f6">Pending</span></td>
					<td>$78.50</td>
				</tr>
			</tbody>
		</table>
		<p><em>2 active orders</em></p>
	`

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Active orders retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetOrder(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service GET /orders/:id

	html := fmt.Sprintf(`
		<p><strong>Order #%s Details:</strong></p>
		<ul>
			<li><strong>Table:</strong> Table 2</li>
			<li><strong>Status:</strong> <span style="color: #f59e0b">Preparing</span></li>
			<li><strong>Server:</strong> Maria</li>
			<li><strong>Items:</strong>
				<ul>
					<li>Burger × 2 - $20.00</li>
					<li>Fries × 2 - $10.00</li>
					<li>Soda × 3 - $15.00</li>
				</ul>
			</li>
			<li><strong>Subtotal:</strong> $45.00</li>
			<li><strong>Tax:</strong> $4.50</li>
			<li><strong>Total:</strong> $49.50</li>
			<li><strong>Created:</strong> 10 minutes ago</li>
		</ul>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Order %s retrieved", orderID),
	}, nil
}

func (p *DeterministicParser) handleGetOrderItems(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service GET /orders/:id/items

	html := fmt.Sprintf(`
		<p><strong>Items in Order #%s:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Item</th>
					<th>Code</th>
					<th>Qty</th>
					<th>Unit Price</th>
					<th>Total</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Burger</td>
					<td>BURG-001</td>
					<td>2</td>
					<td>$10.00</td>
					<td>$20.00</td>
					<td><span style="color: #f59e0b">Preparing</span></td>
				</tr>
				<tr>
					<td>Fries</td>
					<td>SIDE-002</td>
					<td>2</td>
					<td>$5.00</td>
					<td>$10.00</td>
					<td><span style="color: #10b981">Ready</span></td>
				</tr>
				<tr>
					<td>Soda</td>
					<td>DRINK-003</td>
					<td>3</td>
					<td>$5.00</td>
					<td>$15.00</td>
					<td><span style="color: #10b981">Ready</span></td>
				</tr>
			</tbody>
		</table>
		<p><em>3 items, Total: $45.00</em></p>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Order items retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetOrderStatus(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service GET /orders/:id/status

	html := fmt.Sprintf(`
		<p><strong>Order #%s Status:</strong></p>
		<ul>
			<li><strong>Current State:</strong> <span style="color: #f59e0b">Preparing</span></li>
			<li><strong>Progress:</strong> 2/3 items ready</li>
			<li><strong>Estimated Ready:</strong> 5 minutes</li>
			<li><strong>Last Updated:</strong> 2 minutes ago</li>
		</ul>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Order status retrieved",
	}, nil
}

// ORDER COMMANDS

func (p *DeterministicParser) handleOpenOrder(ctx context.Context, params []string) (*CommandResponse, error) {
	tableID := params[0]
	// TODO: Call order service POST /orders with table_id
	// Backend will generate conversational ID using atomic counter
	// and return both conversational_id (e.g., "47") and internal UUID

	// Mock: simulating backend response with conversational ID
	conversationalID := "47" // TODO: Replace with actual backend response
	html := fmt.Sprintf(`
		<p>✅ <strong>Order Opened Successfully</strong></p>
		<ul>
			<li><strong>Order ID:</strong> <code>%s</code></li>
			<li><strong>Table:</strong> Table %s</li>
			<li><strong>Status:</strong> <span style="color: #3b82f6">Pending</span></li>
			<li><strong>Created:</strong> Just now</li>
		</ul>
		<p><em>Use <code>add item %s [item] [qty]</code> to add items</em></p>
	`, conversationalID, tableID, conversationalID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Order %s opened for table %s", conversationalID, tableID),
	}, nil
}

func (p *DeterministicParser) handleCloseOrder(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service POST /orders/:id/close

	html := fmt.Sprintf(`
		<p>✅ <strong>Order #%s Closed</strong></p>
		<ul>
			<li><strong>Final Total:</strong> $49.50</li>
			<li><strong>Payment Status:</strong> Pending</li>
			<li><strong>Closed At:</strong> Just now</li>
		</ul>
		<p><em>Order ready for payment processing</em></p>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Order %s closed", orderID),
	}, nil
}

func (p *DeterministicParser) handleCancelOrder(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service POST /orders/:id/cancel

	html := fmt.Sprintf(`
		<p>⚠️ <strong>Order #%s Cancelled</strong></p>
		<ul>
			<li><strong>Status:</strong> <span style="color: #ef4444">Cancelled</span></li>
			<li><strong>Cancelled At:</strong> Just now</li>
		</ul>
		<p><em>All items removed from kitchen queue</em></p>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Order %s cancelled", orderID),
	}, nil
}

func (p *DeterministicParser) handleAddItem(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	itemCode := params[1]
	quantity := params[2]
	// TODO: Call order service POST /orders/:id/items

	html := fmt.Sprintf(`
		<p>✅ <strong>Item Added to Order</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Item:</strong> %s</li>
			<li><strong>Quantity:</strong> %s</li>
			<li><strong>Status:</strong> Added to order</li>
		</ul>
		<p><em>Use <code>send to kitchen %s</code> to send to kitchen</em></p>
	`, orderID, itemCode, quantity, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Item added successfully",
	}, nil
}

func (p *DeterministicParser) handleRemoveItem(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	itemCode := params[1]
	// TODO: Call order service DELETE /orders/:id/items/:item_code

	html := fmt.Sprintf(`
		<p>✅ <strong>Item Removed from Order</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Item:</strong> %s</li>
			<li><strong>Status:</strong> Removed</li>
		</ul>
	`, orderID, itemCode)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Item removed successfully",
	}, nil
}

func (p *DeterministicParser) handleUpdateItem(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	itemCode := params[1]
	quantity := params[2]
	// TODO: Call order service PATCH /orders/:id/items/:item_code

	html := fmt.Sprintf(`
		<p>✅ <strong>Item Updated</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Item:</strong> %s</li>
			<li><strong>New Quantity:</strong> %s</li>
			<li><strong>Status:</strong> Updated</li>
		</ul>
	`, orderID, itemCode, quantity)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Item updated successfully",
	}, nil
}

func (p *DeterministicParser) handleSendToKitchen(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service POST /orders/:id/send

	html := fmt.Sprintf(`
		<p>✅ <strong>Order Sent to Kitchen</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Status:</strong> <span style="color: #f59e0b">In Kitchen</span></li>
			<li><strong>Estimated Time:</strong> 15-20 minutes</li>
		</ul>
		<p><em>Kitchen has been notified</em></p>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Order sent to kitchen",
	}, nil
}

func (p *DeterministicParser) handleMarkReady(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service POST /orders/:id/ready

	html := fmt.Sprintf(`
		<p>✅ <strong>Order Marked as Ready</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Status:</strong> <span style="color: #10b981">Ready for Delivery</span></li>
			<li><strong>All items:</strong> Prepared</li>
		</ul>
		<p><em>Server can now deliver to table</em></p>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Order marked as ready",
	}, nil
}

func (p *DeterministicParser) handleReopenOrder(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service POST /orders/:id/reopen

	html := fmt.Sprintf(`
		<p>✅ <strong>Order #%s Reopened</strong></p>
		<ul>
			<li><strong>Status:</strong> <span style="color: #3b82f6">Active</span></li>
			<li><strong>Reopened At:</strong> Just now</li>
		</ul>
		<p><em>Order is now active and can be modified</em></p>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Order %s reopened", orderID),
	}, nil
}

func (p *DeterministicParser) handleAddNote(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	note := params[1]
	// TODO: Call order service POST /orders/:id/note

	html := fmt.Sprintf(`
		<p>✅ <strong>Note Added to Order</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Note:</strong> %s</li>
			<li><strong>Added:</strong> Just now</li>
		</ul>
	`, orderID, note)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Note added successfully",
	}, nil
}

func (p *DeterministicParser) handleAssignOrder(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	userID := params[1]
	// TODO: Call order service POST /orders/:id/assign

	html := fmt.Sprintf(`
		<p>✅ <strong>Order Assigned</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Assigned to:</strong> User %s</li>
			<li><strong>Assigned At:</strong> Just now</li>
		</ul>
	`, orderID, userID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Order assigned successfully",
	}, nil
}

func (p *DeterministicParser) handleSplitOrder(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	strategy := params[1]
	// TODO: Call order service POST /orders/:id/split

	html := fmt.Sprintf(`
		<p>✅ <strong>Order Split</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Strategy:</strong> %s</li>
			<li><strong>New Orders:</strong> Creating sub-orders...</li>
		</ul>
		<p><em>Split operation completed</em></p>
	`, orderID, strategy)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Order split successfully",
	}, nil
}

func (p *DeterministicParser) handleMergeOrders(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID1 := params[0]
	orderID2 := params[1]
	// TODO: Call order service POST /orders/merge

	html := fmt.Sprintf(`
		<p>✅ <strong>Orders Merged</strong></p>
		<ul>
			<li><strong>Source Orders:</strong> #%s, #%s</li>
			<li><strong>New Order ID:</strong> 123</li>
			<li><strong>Total Items:</strong> Combined</li>
		</ul>
		<p><em>All items merged into single order</em></p>
	`, orderID1, orderID2)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Orders merged successfully",
	}, nil
}

func (p *DeterministicParser) handleCreateGroup(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	groupLabel := params[1]
	// TODO: Call order service POST /orders/:id/groups

	html := fmt.Sprintf(`
		<p>✅ <strong>Group Created</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Group Label:</strong> %s</li>
			<li><strong>Status:</strong> Empty (ready for items)</li>
		</ul>
		<p><em>Use <code>add item to group %s %s [item] [qty]</code> to add items</em></p>
	`, orderID, groupLabel, orderID, groupLabel)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Group created successfully",
	}, nil
}

func (p *DeterministicParser) handleAddItemToGroup(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	groupLabel := params[1]
	itemCode := params[2]
	quantity := params[3]
	// TODO: Call order service POST /orders/:id/groups/:label/items

	html := fmt.Sprintf(`
		<p>✅ <strong>Item Added to Group</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Group:</strong> %s</li>
			<li><strong>Item:</strong> %s</li>
			<li><strong>Quantity:</strong> %s</li>
		</ul>
	`, orderID, groupLabel, itemCode, quantity)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Item added to group successfully",
	}, nil
}

func (p *DeterministicParser) handleMoveItemToGroup(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	itemCode := params[1]
	targetGroup := params[2]
	// TODO: Call order service PATCH /orders/:id/items/:item_code/move

	html := fmt.Sprintf(`
		<p>✅ <strong>Item Moved to Group</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Item:</strong> %s</li>
			<li><strong>Target Group:</strong> %s</li>
		</ul>
	`, orderID, itemCode, targetGroup)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Item moved successfully",
	}, nil
}

func (p *DeterministicParser) handleRemoveGroup(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	groupLabel := params[1]
	// TODO: Call order service DELETE /orders/:id/groups/:label

	html := fmt.Sprintf(`
		<p>⚠️ <strong>Group Removed</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Group:</strong> %s</li>
			<li><strong>Items:</strong> Moved back to main order</li>
		</ul>
	`, orderID, groupLabel)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Group removed successfully",
	}, nil
}

func (p *DeterministicParser) handleApplyDiscount(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	discount := params[1]
	// TODO: Call order service POST /orders/:id/discount

	html := fmt.Sprintf(`
		<p>✅ <strong>Discount Applied</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>Discount:</strong> %s</li>
			<li><strong>New Total:</strong> Calculating...</li>
		</ul>
	`, orderID, discount)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Discount applied successfully",
	}, nil
}

func (p *DeterministicParser) handleTransferOrder(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	newTableID := params[1]
	// TODO: Call order service PATCH /orders/:id/transfer

	html := fmt.Sprintf(`
		<p>✅ <strong>Order Transferred</strong></p>
		<ul>
			<li><strong>Order:</strong> #%s</li>
			<li><strong>New Table:</strong> Table %s</li>
			<li><strong>Status:</strong> Active</li>
		</ul>
		<p><em>Order moved successfully</em></p>
	`, orderID, newTableID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Order transferred to table %s", newTableID),
	}, nil
}

// ADDITIONAL ORDER QUERIES

func (p *DeterministicParser) handleGetOrdersByTable(ctx context.Context, params []string) (*CommandResponse, error) {
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
					<td>101</td>
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
		Message: "Orders retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetGroups(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service GET /orders/:id/groups

	html := fmt.Sprintf(`
		<p><strong>Groups in Order #%s:</strong></p>
		<table>
			<thead>
				<tr>
					<th>Group Label</th>
					<th>Items</th>
					<th>Subtotal</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td>Person 1</td>
					<td>2</td>
					<td>$25.00</td>
				</tr>
				<tr>
					<td>Person 2</td>
					<td>1</td>
					<td>$20.00</td>
				</tr>
			</tbody>
		</table>
		<p><em>2 groups total</em></p>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Groups retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetOrderHistory(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service GET /orders/:id/history

	html := fmt.Sprintf(`
		<p><strong>Order #%s History:</strong></p>
		<ul>
			<li><strong>Created:</strong> 2025-11-12 10:00 AM by Maria</li>
			<li><strong>Item Added:</strong> Burger × 2 (10:05 AM)</li>
			<li><strong>Sent to Kitchen:</strong> 10:10 AM</li>
			<li><strong>Marked Ready:</strong> 10:25 AM</li>
			<li><strong>Current Status:</strong> <span style="color: #10b981">Ready</span></li>
		</ul>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Order history retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetWaiter(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service GET /orders/:id/waiter

	html := fmt.Sprintf(`
		<p><strong>Assigned Waiter for Order #%s:</strong></p>
		<ul>
			<li><strong>Name:</strong> Maria Rodriguez</li>
			<li><strong>User ID:</strong> USR-123</li>
			<li><strong>Assigned:</strong> 10:00 AM</li>
		</ul>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Waiter info retrieved",
	}, nil
}

func (p *DeterministicParser) handleGetOrderNotes(ctx context.Context, params []string) (*CommandResponse, error) {
	orderID := params[0]
	// TODO: Call order service GET /orders/:id/notes

	html := fmt.Sprintf(`
		<p><strong>Notes for Order #%s:</strong></p>
		<ul>
			<li><strong>[10:05 AM]</strong> Customer allergic to peanuts</li>
			<li><strong>[10:12 AM]</strong> Extra napkins requested</li>
			<li><strong>[10:20 AM]</strong> Rush order - customer in a hurry</li>
		</ul>
		<p><em>3 notes total</em></p>
	`, orderID)

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Order notes retrieved",
	}, nil
}

// UTILITY COMMANDS

func (p *DeterministicParser) handleUndo(ctx context.Context, params []string) (*CommandResponse, error) {
	// TODO: Implement undo logic with command history stack
	// TODO: Determine if last command is reversible
	// TODO: Execute reverse operation
	// TODO: Update command history

	html := `
		<p>⚠️ <strong>Undo Not Yet Implemented</strong></p>
		<div style="background: #fef3c7; padding: 1rem; border-radius: 0.5rem; border-left: 4px solid #f59e0b; margin-top: 0.5rem;">
			<p style="margin: 0;"><strong>Planned Functionality:</strong></p>
			<ul style="margin: 0.5rem 0 0 1.5rem; padding: 0;">
				<li>Maintains command history stack per session</li>
				<li>Determines if last command is reversible</li>
				<li>Confirms action before executing undo</li>
				<li>Shows what will be undone</li>
			</ul>
			<p style="margin: 0.5rem 0 0 0; font-size: 0.9em;"><em>Example: After "add item 47 BURGER 2", undo would remove those items</em></p>
		</div>
		<p><em>This command will be implemented in a future phase</em></p>
	`

	return &CommandResponse{
		HTML:    html,
		Success: false,
		Message: "Undo command not yet implemented",
	}, nil
}
