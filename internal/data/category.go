package data

import "go-kasir-api/internal/model"

// Categories is in-memory storage for category data.
var Categories = []model.Category{
	{ID: 1, Name: "Noodles", Description: "Instant noodles and variants"},
	{ID: 2, Name: "Beverages", Description: "Drinks and bottled water"},
	{ID: 3, Name: "Condiments", Description: "Sauces and seasoning"},
}
