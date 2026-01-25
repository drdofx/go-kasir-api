package data

import "go-kasir-api/internal/model"

// Products is in-memory storage (temporary, will be replaced by a database later).
var Products = []model.Product{
	{ID: 1, Name: "Instant Noodles", Price: 3500, Stock: 10},
	{ID: 2, Name: "Mineral Water 1000ml", Price: 3000, Stock: 40},
	{ID: 3, Name: "Soy Sauce", Price: 12000, Stock: 20},
}
