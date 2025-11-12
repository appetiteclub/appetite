package operations

import (
	"context"
)

func (p *DeterministicParser) handleHelp(ctx context.Context, params []string) (*CommandResponse, error) {
	html := `
		<div style="max-width: 900px;">
			<h3>🎯 Command Reference Guide</h3>
			<p><em>Supports: English | Español | Polski</em></p>

			<h4>📦 Order Management</h4>
			<table style="width: 100%; font-size: 0.85em; margin-bottom: 1.5rem;">
				<thead>
					<tr>
						<th style="text-align: left; width: 30%;">Command</th>
						<th style="text-align: left; width: 15%;">Short</th>
						<th style="text-align: left;">Example</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td><strong>Queries</strong></td>
						<td></td>
						<td></td>
					</tr>
					<tr>
						<td><code>list orders</code></td>
						<td><code>lo</code></td>
						<td>lo | list orders | zamówienia</td>
					</tr>
					<tr>
						<td><code>active orders</code></td>
						<td><code>lao</code></td>
						<td>lao | active orders | ordenes activas</td>
					</tr>
					<tr>
						<td><code>get order</code></td>
						<td><code>go</code></td>
						<td>go 47 | get order 47</td>
					</tr>
					<tr>
						<td><code>order items</code></td>
						<td><code>gi</code></td>
						<td>gi 47 | order items 47</td>
					</tr>
					<tr>
						<td><code>order status</code></td>
						<td><code>gs</code></td>
						<td>gs 47 | status 47</td>
					</tr>
					<tr style="background: #f9fafb;">
						<td><strong>Actions</strong></td>
						<td></td>
						<td></td>
					</tr>
					<tr>
						<td><code>open order</code></td>
						<td><code>oo</code></td>
						<td>oo 5 | open order 5 | abrir orden 5</td>
					</tr>
					<tr>
						<td><code>close order</code></td>
						<td><code>co</code></td>
						<td>co 47 | close order 47</td>
					</tr>
					<tr>
						<td><code>cancel order</code></td>
						<td><code>xo</code></td>
						<td>xo 47 | cancel order 47</td>
					</tr>
					<tr>
						<td><code>add item</code></td>
						<td><code>ai</code></td>
						<td>add item 47 BURGER 2</td>
					</tr>
					<tr>
						<td><code>remove item</code></td>
						<td><code>ri</code></td>
						<td>remove item 47 BURGER</td>
					</tr>
					<tr>
						<td><code>update item</code></td>
						<td><code>ui</code></td>
						<td>update item 47 BURGER 3</td>
					</tr>
					<tr>
						<td><code>send to kitchen</code></td>
						<td><code>sk</code></td>
						<td>sk 47 | send kitchen 47</td>
					</tr>
					<tr>
						<td><code>mark ready</code></td>
						<td><code>mr</code></td>
						<td>mr 47 | ready 47</td>
					</tr>
					<tr style="background: #f0f9ff;">
						<td><strong>Advanced</strong></td>
						<td></td>
						<td></td>
					</tr>
					<tr>
						<td><code>split order</code></td>
						<td><code>so</code></td>
						<td>split order 47 by-person</td>
					</tr>
					<tr>
						<td><code>merge orders</code></td>
						<td><code>mo</code></td>
						<td>merge orders 47 52</td>
					</tr>
					<tr>
						<td><code>create group</code></td>
						<td><code>cg</code></td>
						<td>create group 47 "Group A"</td>
					</tr>
					<tr>
						<td><code>apply discount</code></td>
						<td><code>ad</code></td>
						<td>apply discount 47 10%</td>
					</tr>
					<tr>
						<td><code>transfer order</code></td>
						<td><code>to</code></td>
						<td>transfer order 47 8</td>
					</tr>
				</tbody>
			</table>

			<h4>🪑 Table Management</h4>
			<table style="width: 100%; font-size: 0.85em; margin-bottom: 1.5rem;">
				<thead>
					<tr>
						<th style="text-align: left; width: 30%;">Command</th>
						<th style="text-align: left; width: 15%;">Short</th>
						<th style="text-align: left;">Example</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td><strong>Queries</strong></td>
						<td></td>
						<td></td>
					</tr>
					<tr>
						<td><code>list tables</code></td>
						<td><code>lt</code></td>
						<td>lt | tables | mesas | stoliki</td>
					</tr>
					<tr>
						<td><code>available tables</code></td>
						<td><code>lat</code></td>
						<td>lat | available tables</td>
					</tr>
					<tr>
						<td><code>occupied tables</code></td>
						<td><code>lot</code></td>
						<td>lot | occupied tables</td>
					</tr>
					<tr>
						<td><code>get table</code></td>
						<td><code>gt</code></td>
						<td>gt 5 | table 5 | mesa 5</td>
					</tr>
					<tr>
						<td><code>table status</code></td>
						<td><code>gts</code></td>
						<td>gts 5 | table status 5</td>
					</tr>
					<tr style="background: #f9fafb;">
						<td><strong>Seating & Reservations</strong></td>
						<td></td>
						<td></td>
					</tr>
					<tr>
						<td><code>seat party</code></td>
						<td><code>sp</code></td>
						<td>sp 3 4 | seat party 3 4</td>
					</tr>
					<tr>
						<td><code>release table</code></td>
						<td><code>rt</code></td>
						<td>rt 3 | release table 3</td>
					</tr>
					<tr>
						<td><code>reserve table</code></td>
						<td><code>rv</code></td>
						<td>rv 5 "Smith" | reserve table 5</td>
					</tr>
					<tr>
						<td><code>cancel reservation</code></td>
						<td><code>cr</code></td>
						<td>cr 5 | cancel reservation 5</td>
					</tr>
					<tr style="background: #f0f9ff;">
						<td><strong>Management</strong></td>
						<td></td>
						<td></td>
					</tr>
					<tr>
						<td><code>assign waiter</code></td>
						<td><code>aw</code></td>
						<td>assign waiter 5 USR-123</td>
					</tr>
					<tr>
						<td><code>clean table</code></td>
						<td><code>mtc</code></td>
						<td>mtc 5 | clean table 5</td>
					</tr>
					<tr>
						<td><code>dirty table</code></td>
						<td><code>mtd</code></td>
						<td>mtd 5 | dirty table 5</td>
					</tr>
					<tr>
						<td><code>create table</code></td>
						<td><code>ct</code></td>
						<td>create table 10 6</td>
					</tr>
					<tr>
						<td><code>merge tables</code></td>
						<td><code>mt</code></td>
						<td>merge tables 3 4</td>
					</tr>
					<tr>
						<td><code>block table</code></td>
						<td><code>bt</code></td>
						<td>block table 5 "maintenance"</td>
					</tr>
				</tbody>
			</table>

			<h4>🔧 Utility Commands</h4>
			<table style="width: 100%; font-size: 0.85em; margin-bottom: 1.5rem;">
				<thead>
					<tr>
						<th style="text-align: left; width: 30%;">Command</th>
						<th style="text-align: left; width: 15%;">Short</th>
						<th style="text-align: left;">Example</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td><code>undo</code></td>
						<td><code>u</code>, <code>un</code></td>
						<td>u | undo | deshacer | cofnij</td>
					</tr>
					<tr>
						<td><code>help</code></td>
						<td><code>h</code></td>
						<td>h | help | ayuda | pomoc</td>
					</tr>
				</tbody>
			</table>

			<h4>💡 Quick Start</h4>
			<div style="background: #f0fdf4; padding: 1rem; border-radius: 0.5rem; border-left: 4px solid #10b981; margin-bottom: 1rem;">
				<p style="margin: 0 0 0.5rem 0;"><strong>Common Workflows:</strong></p>
				<ol style="margin: 0; padding-left: 1.5rem; font-size: 0.9em;">
					<li>Seat guests: <code>seat party 3 4</code></li>
					<li>Open order: <code>open order 3</code> → system returns order ID (e.g., 47)</li>
					<li>Add items: <code>add item 47 BURGER 2</code> → <code>send to kitchen 47</code></li>
					<li>Check status: <code>order status 47</code> → <code>mark ready 47</code></li>
					<li>Close out: <code>close order 47</code> → <code>release table 3</code></li>
				</ol>
			</div>

			<h4>🌍 Language Examples</h4>
			<table style="width: 100%; font-size: 0.85em;">
				<thead>
					<tr>
						<th style="text-align: left;">English</th>
						<th style="text-align: left;">Español</th>
						<th style="text-align: left;">Polski</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td><code>list tables</code></td>
						<td><code>listar mesas</code></td>
						<td><code>lista stolików</code></td>
					</tr>
					<tr>
						<td><code>open order 5</code></td>
						<td><code>abrir orden 5</code></td>
						<td><code>otwórz zamówienie 5</code></td>
					</tr>
					<tr>
						<td><code>seat party 3 4</code></td>
						<td><code>sentar 3 4</code></td>
						<td><code>posadź gości 3 4</code></td>
					</tr>
					<tr>
						<td><code>add item 47 BURGER 2</code></td>
						<td><code>agregar item 47 BURGER 2</code></td>
						<td><code>dodaj pozycję 47 BURGER 2</code></td>
					</tr>
					<tr>
						<td><code>send to kitchen 47</code></td>
						<td><code>enviar cocina 47</code></td>
						<td><code>wyślij do kuchni 47</code></td>
					</tr>
					<tr>
						<td><code>help</code></td>
						<td><code>ayuda</code></td>
						<td><code>pomoc</code></td>
					</tr>
				</tbody>
			</table>

			<div style="margin-top: 1.5rem; padding: 1rem; background: #fef3c7; border-radius: 0.5rem; border-left: 4px solid #f59e0b;">
				<p style="margin: 0; font-size: 0.9em;">
					<strong>💡 Pro tip:</strong> Use short forms for speed during busy service!<br>
					Example: <code>sp 3 4</code> → <code>oo 3</code> (returns 47) → <code>ai 47 BURGER 2</code> → <code>sk 47</code>
				</p>
			</div>

			<p style="margin-top: 1.5rem; text-align: center; color: #666; font-size: 0.85em;">
				<em>Type any command to get started • All commands are case-insensitive</em>
			</p>
		</div>
	`

	return &CommandResponse{
		HTML:    html,
		Success: true,
		Message: "Help displayed",
	}, nil
}
