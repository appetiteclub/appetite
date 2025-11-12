package dictionary

import (
	"context"
	"time"
	"fmt"

	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/seed"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Seeds returns all seeds for the Dictionary service.
func Seeds(db *mongo.Database) []seed.Seed {
	return []seed.Seed{
		{
			ID:          "2025-11-05_restaurant_dictionary",
			Description: "Load restaurant management dictionary",
			Run: func(ctx context.Context) error {
				return seedRestaurantDictionary(ctx, db)
			},
		},
	}
}

// seedRestaurantDictionary creates all sets and options for restaurant operations.
func seedRestaurantDictionary(ctx context.Context, db *mongo.Database) error {
	setsCollection := db.Collection("sets")
	optionsCollection := db.Collection("options")

	setIDMap := make(map[string]string)

	// ========================================
	// TABLE STATUS
	// ========================================
	setID_tablestatus_en := uuid.New().String()
	setIDMap["table_status:en"] = setID_tablestatus_en
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "table_status", "locale": "en"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_tablestatus_en,
		"name":        "table_status",
		"locale":      "en",
		"label":       "Table Status",
		"description": "Status of restaurant tables",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	setID_tablestatus_es := uuid.New().String()
	setIDMap["table_status:es"] = setID_tablestatus_es
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "table_status", "locale": "es"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_tablestatus_es,
		"name":        "table_status",
		"locale":      "es",
		"label":       "Estado de Mesa",
		"description": "Estados de las mesas del restaurante",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// ORDER STATUS
	// ========================================
	setID_orderstatus_en := uuid.New().String()
	setIDMap["order_status:en"] = setID_orderstatus_en
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "order_status", "locale": "en"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_orderstatus_en,
		"name":        "order_status",
		"locale":      "en",
		"label":       "Order Status",
		"description": "Status of orders",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	setID_orderstatus_es := uuid.New().String()
	setIDMap["order_status:es"] = setID_orderstatus_es
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "order_status", "locale": "es"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_orderstatus_es,
		"name":        "order_status",
		"locale":      "es",
		"label":       "Estado de Orden",
		"description": "Estados de las órdenes",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// ORDER ITEM STATUS
	// ========================================
	setID_orderitemstatus_en := uuid.New().String()
	setIDMap["order_item_status:en"] = setID_orderitemstatus_en
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "order_item_status", "locale": "en"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_orderitemstatus_en,
		"name":        "order_item_status",
		"locale":      "en",
		"label":       "Order Item Status",
		"description": "Status of individual order items",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	setID_orderitemstatus_es := uuid.New().String()
	setIDMap["order_item_status:es"] = setID_orderitemstatus_es
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "order_item_status", "locale": "es"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_orderitemstatus_es,
		"name":        "order_item_status",
		"locale":      "es",
		"label":       "Estado de Ítem",
		"description": "Estados de ítems individuales",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// PAYMENT METHOD
	// ========================================
	setID_paymentmethod_en := uuid.New().String()
	setIDMap["payment_method:en"] = setID_paymentmethod_en
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "payment_method", "locale": "en"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_paymentmethod_en,
		"name":        "payment_method",
		"locale":      "en",
		"label":       "Payment Method",
		"description": "Available payment methods",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	setID_paymentmethod_es := uuid.New().String()
	setIDMap["payment_method:es"] = setID_paymentmethod_es
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "payment_method", "locale": "es"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_paymentmethod_es,
		"name":        "payment_method",
		"locale":      "es",
		"label":       "Método de Pago",
		"description": "Métodos de pago disponibles",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// DISH CATEGORY
	// ========================================
	setID_dishcategory_en := uuid.New().String()
	setIDMap["dish_category:en"] = setID_dishcategory_en
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "dish_category", "locale": "en"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_dishcategory_en,
		"name":        "dish_category",
		"locale":      "en",
		"label":       "Dish Category",
		"description": "Categories of dishes",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	setID_dishcategory_es := uuid.New().String()
	setIDMap["dish_category:es"] = setID_dishcategory_es
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "dish_category", "locale": "es"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_dishcategory_es,
		"name":        "dish_category",
		"locale":      "es",
		"label":       "Categoría de Plato",
		"description": "Categorías de platos",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// RESERVATION STATUS
	// ========================================
	setID_reservationstatus_en := uuid.New().String()
	setIDMap["reservation_status:en"] = setID_reservationstatus_en
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "reservation_status", "locale": "en"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_reservationstatus_en,
		"name":        "reservation_status",
		"locale":      "en",
		"label":       "Reservation Status",
		"description": "Status of reservations",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	setID_reservationstatus_es := uuid.New().String()
	setIDMap["reservation_status:es"] = setID_reservationstatus_es
	_, _ = setsCollection.UpdateOne(ctx, bson.M{"name": "reservation_status", "locale": "es"}, bson.M{"$setOnInsert": bson.M{
		"_id":         setID_reservationstatus_es,
		"name":        "reservation_status",
		"locale":      "es",
		"label":       "Estado de Reserva",
		"description": "Estados de las reservas",
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR table_status (EN)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "en", "value": "available"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "en",
		"label":       "Available",
		"value":       "available",
		"description": "Table is available for seating",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "en", "value": "open"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "en",
		"label":       "Open",
		"value":       "open",
		"description": "Table is currently occupied",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "en", "value": "reserved"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "en",
		"label":       "Reserved",
		"value":       "reserved",
		"description": "Table is reserved for future seating",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "en", "value": "cleaning"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "en",
		"label":       "Cleaning",
		"value":       "cleaning",
		"description": "Table is being cleaned",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "en", "value": "out_of_service"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "en",
		"label":       "Out of Service",
		"value":       "out_of_service",
		"description": "Table is not available for use",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR table_status (ES)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "es", "value": "available"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "es",
		"label":       "Disponible",
		"value":       "available",
		"description": "Mesa disponible para sentar clientes",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "es", "value": "open"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "es",
		"label":       "Ocupada",
		"value":       "open",
		"description": "Mesa actualmente ocupada",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "es", "value": "reserved"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "es",
		"label":       "Reservada",
		"value":       "reserved",
		"description": "Mesa reservada para uso futuro",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "es", "value": "cleaning"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "es",
		"label":       "Limpieza",
		"value":       "cleaning",
		"description": "Mesa en proceso de limpieza",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "table_status", "locale": "es", "value": "out_of_service"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "table_status",
		"locale":      "es",
		"label":       "Fuera de Servicio",
		"value":       "out_of_service",
		"description": "Mesa no disponible para uso",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR order_status (EN)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "en", "value": "pending"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "en",
		"label":       "Pending",
		"value":       "pending",
		"description": "Order has been created but not sent to kitchen",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "en", "value": "preparing"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "en",
		"label":       "Preparing",
		"value":       "preparing",
		"description": "Order is being prepared in the kitchen",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "en", "value": "ready"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "en",
		"label":       "Ready",
		"value":       "ready",
		"description": "Order is ready for delivery",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "en", "value": "delivered"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "en",
		"label":       "Delivered",
		"value":       "delivered",
		"description": "Order has been delivered to the table",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "en", "value": "cancelled"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "en",
		"label":       "Cancelled",
		"value":       "cancelled",
		"description": "Order has been cancelled",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR order_status (ES)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "es", "value": "pending"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "es",
		"label":       "Pendiente",
		"value":       "pending",
		"description": "Orden creada pero no enviada a cocina",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "es", "value": "preparing"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "es",
		"label":       "Preparando",
		"value":       "preparing",
		"description": "Orden en preparación en cocina",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "es", "value": "ready"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "es",
		"label":       "Lista",
		"value":       "ready",
		"description": "Orden lista para entrega",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "es", "value": "delivered"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "es",
		"label":       "Entregada",
		"value":       "delivered",
		"description": "Orden entregada a la mesa",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_status", "locale": "es", "value": "cancelled"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_status",
		"locale":      "es",
		"label":       "Cancelada",
		"value":       "cancelled",
		"description": "Orden cancelada",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR order_item_status (EN)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "en", "value": "pending"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "en",
		"label":       "Pending",
		"value":       "pending",
		"description": "Item waiting to be prepared",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "en", "value": "preparing"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "en",
		"label":       "Preparing",
		"value":       "preparing",
		"description": "Item is being prepared",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "en", "value": "ready"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "en",
		"label":       "Ready",
		"value":       "ready",
		"description": "Item is ready for delivery",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "en", "value": "delivered"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "en",
		"label":       "Delivered",
		"value":       "delivered",
		"description": "Item has been delivered to table",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "en", "value": "cancelled"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "en",
		"label":       "Cancelled",
		"value":       "cancelled",
		"description": "Item has been cancelled",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR order_item_status (ES)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "es", "value": "pending"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "es",
		"label":       "Pendiente",
		"value":       "pending",
		"description": "Ítem esperando preparación",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "es", "value": "preparing"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "es",
		"label":       "Preparando",
		"value":       "preparing",
		"description": "Ítem en preparación",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "es", "value": "ready"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "es",
		"label":       "Listo",
		"value":       "ready",
		"description": "Ítem listo para entrega",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "es", "value": "delivered"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "es",
		"label":       "Entregado",
		"value":       "delivered",
		"description": "Ítem entregado a la mesa",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "order_item_status", "locale": "es", "value": "cancelled"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "order_item_status",
		"locale":      "es",
		"label":       "Cancelado",
		"value":       "cancelled",
		"description": "Ítem cancelado",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR payment_method (EN)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "en", "value": "cash"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "en",
		"label":       "Cash",
		"value":       "cash",
		"description": "Payment in cash",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "en", "value": "credit_card"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "en",
		"label":       "Credit Card",
		"value":       "credit_card",
		"description": "Payment by credit card",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "en", "value": "debit_card"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "en",
		"label":       "Debit Card",
		"value":       "debit_card",
		"description": "Payment by debit card",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "en", "value": "mobile_payment"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "en",
		"label":       "Mobile Payment",
		"value":       "mobile_payment",
		"description": "Payment via mobile app",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "en", "value": "voucher"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "en",
		"label":       "Voucher",
		"value":       "voucher",
		"description": "Payment by voucher or coupon",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR payment_method (ES)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "es", "value": "cash"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "es",
		"label":       "Efectivo",
		"value":       "cash",
		"description": "Pago en efectivo",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "es", "value": "credit_card"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "es",
		"label":       "Tarjeta de Crédito",
		"value":       "credit_card",
		"description": "Pago con tarjeta de crédito",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "es", "value": "debit_card"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "es",
		"label":       "Tarjeta de Débito",
		"value":       "debit_card",
		"description": "Pago con tarjeta de débito",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "es", "value": "mobile_payment"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "es",
		"label":       "Pago Móvil",
		"value":       "mobile_payment",
		"description": "Pago vía aplicación móvil",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "payment_method", "locale": "es", "value": "voucher"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "payment_method",
		"locale":      "es",
		"label":       "Vale",
		"value":       "voucher",
		"description": "Pago con vale o cupón",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR dish_category (EN)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "en", "value": "appetizer"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "en",
		"label":       "Appetizer",
		"value":       "appetizer",
		"description": "Starter dishes",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "en", "value": "main_course"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "en",
		"label":       "Main Course",
		"value":       "main_course",
		"description": "Main dishes",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "en", "value": "side_dish"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "en",
		"label":       "Side Dish",
		"value":       "side_dish",
		"description": "Side accompaniments",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "en", "value": "dessert"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "en",
		"label":       "Dessert",
		"value":       "dessert",
		"description": "Sweet dishes",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "en", "value": "beverage"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "en",
		"label":       "Beverage",
		"value":       "beverage",
		"description": "Non-alcoholic drinks",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "en", "value": "alcohol"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "en",
		"label":       "Alcohol",
		"value":       "alcohol",
		"description": "Alcoholic beverages",
		"position":    6,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR dish_category (ES)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "es", "value": "appetizer"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "es",
		"label":       "Entrada",
		"value":       "appetizer",
		"description": "Platos de entrada",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "es", "value": "main_course"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "es",
		"label":       "Plato Principal",
		"value":       "main_course",
		"description": "Platos principales",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "es", "value": "side_dish"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "es",
		"label":       "Acompañamiento",
		"value":       "side_dish",
		"description": "Acompañamientos",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "es", "value": "dessert"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "es",
		"label":       "Postre",
		"value":       "dessert",
		"description": "Postres",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "es", "value": "beverage"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "es",
		"label":       "Bebida",
		"value":       "beverage",
		"description": "Bebidas sin alcohol",
		"position":    5,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "dish_category", "locale": "es", "value": "alcohol"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "dish_category",
		"locale":      "es",
		"label":       "Alcohol",
		"value":       "alcohol",
		"description": "Bebidas alcohólicas",
		"position":    6,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR reservation_status (EN)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "reservation_status", "locale": "en", "value": "confirmed"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "reservation_status",
		"locale":      "en",
		"label":       "Confirmed",
		"value":       "confirmed",
		"description": "Reservation is confirmed",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "reservation_status", "locale": "en", "value": "seated"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "reservation_status",
		"locale":      "en",
		"label":       "Seated",
		"value":       "seated",
		"description": "Guest has been seated",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "reservation_status", "locale": "en", "value": "cancelled"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "reservation_status",
		"locale":      "en",
		"label":       "Cancelled",
		"value":       "cancelled",
		"description": "Reservation has been cancelled",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "reservation_status", "locale": "en", "value": "no_show"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "reservation_status",
		"locale":      "en",
		"label":       "No Show",
		"value":       "no_show",
		"description": "Guest did not arrive",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	// ========================================
	// OPTIONS FOR reservation_status (ES)
	// ========================================
	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "reservation_status", "locale": "es", "value": "confirmed"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "reservation_status",
		"locale":      "es",
		"label":       "Confirmada",
		"value":       "confirmed",
		"description": "Reserva confirmada",
		"position":    1,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "reservation_status", "locale": "es", "value": "seated"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "reservation_status",
		"locale":      "es",
		"label":       "Sentado",
		"value":       "seated",
		"description": "Cliente ya sentado",
		"position":    2,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "reservation_status", "locale": "es", "value": "cancelled"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "reservation_status",
		"locale":      "es",
		"label":       "Cancelada",
		"value":       "cancelled",
		"description": "Reserva cancelada",
		"position":    3,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	_, _ = optionsCollection.UpdateOne(ctx, bson.M{"set_name": "reservation_status", "locale": "es", "value": "no_show"}, bson.M{"$setOnInsert": bson.M{
		"_id":         uuid.New().String(),
		"set_name":    "reservation_status",
		"locale":      "es",
		"label":       "No Llegó",
		"value":       "no_show",
		"description": "Cliente no llegó",
		"position":    4,
		"active":      true,
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"created_by":  "system",
		"updated_by":  "system",
	}}, options.Update().SetUpsert(true))

	return nil
}


// SeedingFunc returns a function that applies database seeds for the dictionary service.
// dbFn is a function that returns the mongo database; it's invoked at runtime so callers
// (like main) can pass a closure that reads the database from a repo that is started
// by the lifecycle before OnStart runs.
func SeedingFunc(appName string, dbFn func() *mongo.Database, logger aqm.Logger) func(ctx context.Context) error {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}

	return func(ctx context.Context) error {
		logger.Info("Applying database seeds...")
		db := dbFn()
		if db == nil {
			return fmt.Errorf("database is not initialized")
		}
		tracker := seed.NewMongoTracker(db)
		seeds := Seeds(db)
		if err := seed.Apply(ctx, tracker, seeds, appName); err != nil {
			return fmt.Errorf("apply seeds: %w", err)
		}
		logger.Info("Database seeds applied successfully")
		return nil
	}
}
