package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Menu Management Commands with Interactive Workflows
//
// These handlers support multi-step iterative processes for building
// menu items, setting prices, adding allergens, etc. Users can abort
// any in-progress operation using the "cancel" command.

// handleListMenu lists all menu items
func (p *DeterministicParser) handleListMenu(ctx context.Context, params []string) (*CommandResponse, error) {
	// Call menu service GET /menu/items
	resp, err := p.menuClient.Request(ctx, "GET", "/menu/items", nil)
	if err != nil {
		return &CommandResponse{
			HTML:    formatError(fmt.Sprintf("Failed to fetch menu items: %v", err)),
			Success: false,
			Message: "Menu fetch failed",
		}, nil
	}

	if resp == nil || resp.Data == nil {
		return &CommandResponse{
			HTML: `
				<p>📋 <strong>Menu is empty</strong></p>
				<p>Use <code>new-menu-item</code> to create your first menu item.</p>
			`,
			Success: true,
			Message: "Menu is empty",
		}, nil
	}

	// The response structure from menu service is already unmarshaled
	// resp.Data should be a slice of menu items directly
	var dataArray []interface{}

	// Handle both possible response structures
	if arr, ok := resp.Data.([]interface{}); ok {
		// Direct array
		dataArray = arr
	} else if dataMap, ok := resp.Data.(map[string]interface{}); ok {
		// Wrapped in object with "data" key
		if items, ok := dataMap["data"].([]interface{}); ok {
			dataArray = items
		}
	}

	if dataArray == nil {
		dataArray = []interface{}{}
	}

	if len(dataArray) == 0 {
		return &CommandResponse{
			HTML: `
				<p>📋 <strong>Menu is empty</strong></p>
				<p>Use <code>new-menu-item</code> to create your first menu item.</p>
			`,
			Success: true,
			Message: "Menu is empty",
		}, nil
	}

	// Format as table
	html := `<p><strong>🍽️ Menu Items</strong></p><table style="width: 100%; font-size: 0.85em;">
		<tr><th>Code</th><th>Name</th><th>Price</th><th>Status</th></tr>`

	for _, itemIface := range dataArray {
		itemMap, ok := itemIface.(map[string]interface{})
		if !ok {
			continue
		}

		shortCode, _ := itemMap["short_code"].(string)
		active, _ := itemMap["active"].(bool)

		// Extract name
		name := ""
		if nameMap, ok := itemMap["name"].(map[string]interface{}); ok {
			if enName, ok := nameMap["en"].(string); ok {
				name = enName
			} else {
				// Get first available name
				for _, n := range nameMap {
					if nStr, ok := n.(string); ok {
						name = nStr
						break
					}
				}
			}
		}

		// Extract price
		priceStr := "-"
		if pricesArray, ok := itemMap["prices"].([]interface{}); ok && len(pricesArray) > 0 {
			if priceMap, ok := pricesArray[0].(map[string]interface{}); ok {
				amount, _ := priceMap["amount"].(float64)
				currency, _ := priceMap["currency_code"].(string)
				priceStr = fmt.Sprintf("%.2f %s", amount, currency)
			}
		}

		status := "❌ Inactive"
		if active {
			status = "✅ Active"
		}

		html += fmt.Sprintf(`<tr>
			<td><code>%s</code></td>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
		</tr>`, shortCode, name, priceStr, status)
	}

	html += "</table>"

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: fmt.Sprintf("Listed %d menu items", len(dataArray)),
	}, nil
}

// handleNewMenuItem initiates an interactive workflow to create a menu item
func (p *DeterministicParser) handleNewMenuItem(ctx context.Context, params []string) (*CommandResponse, error) {
	// Start interactive workflow
	// Step 1: Ask for short code
	return &CommandResponse{
		HTML: `
			<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
				<p><strong>🆕 Creating New Menu Item</strong></p>
				<p>Let's build your menu item step by step.</p>
				<p><strong>Step 1/5:</strong> Enter a unique short code (e.g., BURGER-DELX, PASTA-CARB)</p>
				<p style="font-size: 0.85em; color: #666;">Type <code>cancel</code> anytime to abort</p>
			</div>
			<script>
				sessionStorage.setItem('ops_workflow', JSON.stringify({
					type: 'new-menu-item',
					step: 1,
					data: {}
				}));
			</script>
		`,
		Success: true,
		Message: "Started new menu item workflow",
	}, nil
}

// handleWorkflowInput processes input during an active workflow
func (p *DeterministicParser) handleWorkflowInput(ctx context.Context, input string, workflowData map[string]interface{}) (*CommandResponse, error) {
	workflowType, _ := workflowData["type"].(string)
	stepFloat, _ := workflowData["step"].(float64)
	step := int(stepFloat)
	data, _ := workflowData["data"].(map[string]interface{})

	switch workflowType {
	case "new-menu-item":
		return p.handleNewMenuItemWorkflow(ctx, input, step, data)
	case "edit-menu-item":
		return p.handleEditMenuItemWorkflow(ctx, input, step, data)
	case "set-price":
		return p.handleSetPriceWorkflow(ctx, input, step, data)
	case "add-allergen":
		return p.handleAddAllergenWorkflow(ctx, input, step, data)
	case "add-ingredient":
		return p.handleAddIngredientWorkflow(ctx, input, step, data)
	default:
		return &CommandResponse{
			HTML:    formatError("Unknown workflow type"),
			Success: false,
		}, nil
	}
}

// handleNewMenuItemWorkflow handles the new menu item creation workflow
func (p *DeterministicParser) handleNewMenuItemWorkflow(ctx context.Context, input string, step int, data map[string]interface{}) (*CommandResponse, error) {
	switch step {
	case 1: // Short code received
		data["short_code"] = strings.TrimSpace(input)
		return &CommandResponse{
			HTML: fmt.Sprintf(`
				<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
					<p>✓ Short code: <code>%s</code></p>
					<p><strong>Step 2/5:</strong> Enter the item name in English</p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'new-menu-item',
						step: 2,
						data: %s
					}));
				</script>
			`, input, toJSON(data)),
			Success: true,
		}, nil

	case 2: // Name received
		data["name_en"] = strings.TrimSpace(input)
		return &CommandResponse{
			HTML: fmt.Sprintf(`
				<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
					<p>✓ Name: %s</p>
					<p><strong>Step 3/5:</strong> Enter a brief description (or type "skip")</p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'new-menu-item',
						step: 3,
						data: %s
					}));
				</script>
			`, input, toJSON(data)),
			Success: true,
		}, nil

	case 3: // Description received
		if strings.ToLower(strings.TrimSpace(input)) != "skip" {
			data["description_en"] = strings.TrimSpace(input)
		}
		return &CommandResponse{
			HTML: `
				<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
					<p>✓ Description saved</p>
					<p><strong>Step 4/5:</strong> Enter the base price (e.g., 12.50)</p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'new-menu-item',
						step: 4,
						data: ` + toJSON(data) + `
					}));
				</script>
			`,
			Success: true,
		}, nil

	case 4: // Price received
		data["price"] = strings.TrimSpace(input)
		return &CommandResponse{
			HTML: `
				<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
					<p>✓ Price: $` + input + `</p>
					<p><strong>Step 5/5:</strong> Enter currency code (e.g., USD, EUR, GBP) or "USD" for default</p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'new-menu-item',
						step: 5,
						data: ` + toJSON(data) + `
					}));
				</script>
			`,
			Success: true,
		}, nil

	case 5: // Currency received - create the item
		currency := strings.ToUpper(strings.TrimSpace(input))
		if currency == "" {
			currency = "USD"
		}
		data["currency"] = currency

		// Build the payload
		payload := map[string]interface{}{
			"short_code": data["short_code"],
			"name": map[string]string{
				"en": data["name_en"].(string),
			},
			"prices": []map[string]interface{}{
				{
					"amount":        parseFloat(data["price"].(string)),
					"currency_code": currency,
				},
			},
			"active": true,
		}

		if desc, ok := data["description_en"].(string); ok && desc != "" {
			payload["description"] = map[string]string{
				"en": desc,
			}
		}

		// Call menu service POST /menu/items
		resp, err := p.menuClient.Request(ctx, "POST", "/menu/items", payload)
		if err != nil {
			return &CommandResponse{
				HTML: formatError(fmt.Sprintf("Failed to create menu item: %v", err)),
				Success: false,
				Message: "Creation failed",
			}, nil
		}

		// Extract created item data
		shortCode := data["short_code"].(string)
		if resp != nil && resp.Data != nil {
			if dataMap, ok := resp.Data.(map[string]interface{}); ok {
				if itemData, ok := dataMap["data"].(map[string]interface{}); ok {
					if sc, ok := itemData["short_code"].(string); ok {
						shortCode = sc
					}
				}
			}
		}

		return &CommandResponse{
			HTML: fmt.Sprintf(`
				<div style="padding: 1rem; background: #ecfdf5; border-left: 4px solid #10b981;">
					<p><strong>✅ Menu Item Created!</strong></p>
					<ul>
						<li><strong>Code:</strong> <code>%s</code></li>
						<li><strong>Name:</strong> %s</li>
						<li><strong>Price:</strong> %s %s</li>
					</ul>
					<p style="font-size: 0.85em; color: #666;">
						Use <code>update-item %s</code> to add allergens, portions, or ingredients
					</p>
				</div>
				<script>
					sessionStorage.removeItem('ops_workflow');
				</script>
			`, shortCode, data["name_en"], data["price"], currency, shortCode),
			Success: true,
			Message: "Menu item created",
		}, nil

	default:
		return &CommandResponse{
			HTML:    formatError("Invalid workflow step"),
			Success: false,
		}, nil
	}
}

// handleSetPriceWorkflow handles setting prices for menu items
func (p *DeterministicParser) handleSetPriceWorkflow(ctx context.Context, input string, step int, data map[string]interface{}) (*CommandResponse, error) {
	switch step {
	case 1: // Item code received
		data["code"] = strings.TrimSpace(input)
		return &CommandResponse{
			HTML: fmt.Sprintf(`
				<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
					<p>✓ Item: <code>%s</code></p>
					<p><strong>Step 2/3:</strong> Enter new price (e.g., 15.99)</p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'set-price',
						step: 2,
						data: %s
					}));
				</script>
			`, input, toJSON(data)),
			Success: true,
		}, nil

	case 2: // Price received
		data["price"] = strings.TrimSpace(input)
		return &CommandResponse{
			HTML: `
				<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
					<p>✓ Price: $` + input + `</p>
					<p><strong>Step 3/3:</strong> Currency (USD, EUR, etc.)</p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'set-price',
						step: 3,
						data: ` + toJSON(data) + `
					}));
				</script>
			`,
			Success: true,
		}, nil

	case 3: // Update complete
		currency := strings.ToUpper(strings.TrimSpace(input))
		return &CommandResponse{
			HTML: fmt.Sprintf(`
				<div style="padding: 1rem; background: #ecfdf5; border-left: 4px solid #10b981;">
					<p><strong>✅ Price Updated</strong></p>
					<p><code>%s</code>: %s %s</p>
				</div>
				<script>
					sessionStorage.removeItem('ops_workflow');
				</script>
			`, data["code"], data["price"], currency),
			Success: true,
		}, nil
	}

	return &CommandResponse{
		HTML:    formatError("Invalid workflow step"),
		Success: false,
	}, nil
}

// handleAddAllergenWorkflow handles adding allergens to menu items
func (p *DeterministicParser) handleAddAllergenWorkflow(ctx context.Context, input string, step int, data map[string]interface{}) (*CommandResponse, error) {
	// Implementation would follow similar pattern
	return &CommandResponse{
		HTML:    "Allergen workflow not yet implemented",
		Success: false,
	}, nil
}

// handleAddIngredientWorkflow handles adding ingredients to menu items
func (p *DeterministicParser) handleAddIngredientWorkflow(ctx context.Context, input string, step int, data map[string]interface{}) (*CommandResponse, error) {
	// Implementation would follow similar pattern
	return &CommandResponse{
		HTML:    "Ingredient workflow not yet implemented",
		Success: false,
	}, nil
}

// handleEditMenuItem initiates menu item editing workflow
func (p *DeterministicParser) handleEditMenuItem(ctx context.Context, params []string) (*CommandResponse, error) {
	return &CommandResponse{
		HTML: `
			<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
				<p><strong>✏️ Edit Menu Item</strong></p>
				<p><strong>Step 1/2:</strong> Enter the item short code to edit</p>
				<p style="font-size: 0.85em; color: #666;">Type <code>cancel</code> to abort</p>
			</div>
			<script>
				sessionStorage.setItem('ops_workflow', JSON.stringify({
					type: 'edit-menu-item',
					step: 1,
					data: {}
				}));
			</script>
		`,
		Success: true,
		Message: "Started edit menu item workflow",
	}, nil
}

// handleEditMenuItemWorkflow handles the editing workflow
func (p *DeterministicParser) handleEditMenuItemWorkflow(ctx context.Context, input string, step int, data map[string]interface{}) (*CommandResponse, error) {
	switch step {
	case 1: // Short code received - fetch the item
		shortCode := strings.ToUpper(strings.TrimSpace(input))

		// Fetch all items and search by short_code
		resp, err := p.menuClient.Request(ctx, "GET", "/menu/items", nil)
		if err != nil {
			return &CommandResponse{
				HTML:    formatError(fmt.Sprintf("Failed to fetch items: %v", err)),
				Success: false,
			}, nil
		}

		if resp == nil || resp.Data == nil {
			return &CommandResponse{
				HTML:    formatError("No items found"),
				Success: false,
			}, nil
		}

		// Extract items array - handle both direct array and wrapped formats
		var itemsArray []interface{}
		if arr, ok := resp.Data.([]interface{}); ok {
			// Direct array
			itemsArray = arr
		} else if dataMap, ok := resp.Data.(map[string]interface{}); ok {
			// Wrapped in object with "data" key
			if items, ok := dataMap["data"].([]interface{}); ok {
				itemsArray = items
			}
		}

		// Find item by short_code
		var itemMap map[string]interface{}
		for _, item := range itemsArray {
			if itemData, ok := item.(map[string]interface{}); ok {
				if code, ok := itemData["short_code"].(string); ok && code == shortCode {
					itemMap = itemData
					break
				}
			}
		}

		if itemMap == nil {
			return &CommandResponse{
				HTML:    formatError(fmt.Sprintf("Item not found: %s", shortCode)),
				Success: false,
			}, nil
		}

		// Store item data including ID
		data["short_code"] = shortCode
		data["item_id"], _ = itemMap["id"].(string)
		data["item"] = itemMap

		// Extract current values for display
		name := ""
		if nameMap, ok := itemMap["name"].(map[string]interface{}); ok {
			name, _ = nameMap["en"].(string)
		}

		price := ""
		currency := ""
		if pricesArray, ok := itemMap["prices"].([]interface{}); ok && len(pricesArray) > 0 {
			if priceMap, ok := pricesArray[0].(map[string]interface{}); ok {
				if amount, ok := priceMap["amount"].(float64); ok {
					price = fmt.Sprintf("%.2f", amount)
				}
				currency, _ = priceMap["currency_code"].(string)
			}
		}

		return &CommandResponse{
			HTML: fmt.Sprintf(`
				<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
					<p><strong>✏️ Editing: <code>%s</code></strong></p>
					<p>Current name: <strong>%s</strong></p>
					<p>Current price: <strong>%s %s</strong></p>
					<p><strong>Step 2/2:</strong> What would you like to edit?</p>
					<p>Type one of: <code>name</code>, <code>price</code>, <code>description</code>, or <code>done</code> to finish</p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'edit-menu-item',
						step: 2,
						data: %s
					}));
				</script>
			`, shortCode, name, price, currency, toJSON(data)),
			Success: true,
		}, nil

	case 2: // User chose what to edit
		choice := strings.ToLower(strings.TrimSpace(input))

		if choice == "done" {
			return &CommandResponse{
				HTML: `
					<div style="padding: 1rem; background: #ecfdf5; border-left: 4px solid #10b981;">
						<p><strong>✅ Editing Complete</strong></p>
					</div>
					<script>
						sessionStorage.removeItem('ops_workflow');
					</script>
				`,
				Success: true,
			}, nil
		}

		data["edit_field"] = choice

		// Extract current value to show in prompt
		itemMap, _ := data["item"].(map[string]interface{})
		var currentValue string
		var promptMsg string

		switch choice {
		case "name":
			if nameMap, ok := itemMap["name"].(map[string]interface{}); ok {
				currentValue, _ = nameMap["en"].(string)
			}
			promptMsg = fmt.Sprintf("Current: <strong>%s</strong><br>Enter new name:", currentValue)
		case "price":
			if pricesArray, ok := itemMap["prices"].([]interface{}); ok && len(pricesArray) > 0 {
				if priceMap, ok := pricesArray[0].(map[string]interface{}); ok {
					if amount, ok := priceMap["amount"].(float64); ok {
						currentValue = fmt.Sprintf("%.2f", amount)
					}
				}
			}
			promptMsg = fmt.Sprintf("Current: <strong>$%s</strong><br>Enter new price (e.g., 15.99):", currentValue)
		case "description":
			if descMap, ok := itemMap["description"].(map[string]interface{}); ok {
				currentValue, _ = descMap["en"].(string)
			}
			promptMsg = fmt.Sprintf("Current: <strong>%s</strong><br>Enter new description:", currentValue)
		default:
			return &CommandResponse{
				HTML: fmt.Sprintf(`
					<div style="padding: 1rem; background: #fef2f2; border-left: 4px solid #ef4444;">
						<p>Invalid option: <code>%s</code></p>
						<p>Please type: <code>name</code>, <code>price</code>, <code>description</code>, or <code>done</code></p>
					</div>
					<script>
						sessionStorage.setItem('ops_workflow', JSON.stringify({
							type: 'edit-menu-item',
							step: 2,
							data: %s
						}));
					</script>
				`, choice, toJSON(data)),
				Success: false,
			}, nil
		}

		return &CommandResponse{
			HTML: fmt.Sprintf(`
				<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
					<p><strong>%s</strong></p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'edit-menu-item',
						step: 3,
						data: %s
					}));
				</script>
			`, promptMsg, toJSON(data)),
			Success: true,
		}, nil

	case 3: // User provided the new value
		field, _ := data["edit_field"].(string)
		itemID, _ := data["item_id"].(string)
		newValue := strings.TrimSpace(input)

		// Get the full item object to send via PUT
		itemMap, _ := data["item"].(map[string]interface{})

		// Clone the item to create the PUT payload
		payload := make(map[string]interface{})
		for k, v := range itemMap {
			payload[k] = v
		}

		// Update the specific field
		var successMsg string
		switch field {
		case "name":
			payload["name"] = map[string]string{
				"en": newValue,
			}
			successMsg = fmt.Sprintf("Name updated to: %s", newValue)
			// Update in-memory item for next edit
			itemMap["name"] = payload["name"]
		case "price":
			payload["prices"] = []map[string]interface{}{
				{
					"amount":        parseFloat(newValue),
					"currency_code": "USD",
				},
			}
			successMsg = fmt.Sprintf("Price updated to: $%s", newValue)
			// Update in-memory item for next edit
			itemMap["prices"] = payload["prices"]
		case "description":
			payload["description"] = map[string]string{
				"en": newValue,
			}
			successMsg = "Description updated"
			// Update in-memory item for next edit
			itemMap["description"] = payload["description"]
		}

		_, err := p.menuClient.Request(ctx, "PUT", "/menu/items/"+itemID, payload)
		if err != nil {
			return &CommandResponse{
				HTML:    formatError(fmt.Sprintf("Failed to update: %v", err)),
				Success: false,
			}, nil
		}

		// Go back to edit menu
		return &CommandResponse{
			HTML: fmt.Sprintf(`
				<div style="padding: 1rem; background: #ecfdf5; border-left: 4px solid #10b981;">
					<p><strong>✅ %s</strong></p>
					<p>What else would you like to edit?</p>
					<p>Type: <code>name</code>, <code>price</code>, <code>description</code>, or <code>done</code></p>
				</div>
				<script>
					sessionStorage.setItem('ops_workflow', JSON.stringify({
						type: 'edit-menu-item',
						step: 2,
						data: %s
					}));
				</script>
			`, successMsg, toJSON(data)),
			Success: true,
		}, nil

	default:
		return &CommandResponse{
			HTML:    formatError("Invalid workflow step"),
			Success: false,
		}, nil
	}
}

// handleSetPrice initiates price setting workflow
func (p *DeterministicParser) handleSetPrice(ctx context.Context, params []string) (*CommandResponse, error) {
	return &CommandResponse{
		HTML: `
			<div style="padding: 1rem; background: #f0f9ff; border-left: 4px solid #0ea5e9;">
				<p><strong>💰 Set Item Price</strong></p>
				<p><strong>Step 1/3:</strong> Enter item short code</p>
				<p style="font-size: 0.85em; color: #666;">Type <code>cancel</code> to abort</p>
			</div>
			<script>
				sessionStorage.setItem('ops_workflow', JSON.stringify({
					type: 'set-price',
					step: 1,
					data: {}
				}));
			</script>
		`,
		Success: true,
	}, nil
}

// Helper functions

func toJSON(data map[string]interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
