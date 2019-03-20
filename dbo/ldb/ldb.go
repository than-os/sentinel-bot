package ldb

import "github.com/than-os/sentinel-bot/dbo/models"

type BotDB interface {
	// Insert is used to store a new key-value pair
	Insert(string, string, string) error
	// Update is used to update an existing key-value pair
	Update(string, string, string) error
	// Delete would remove one key-pair from the database
	Delete(string, string) error
	// Read would return a key-value pair for a query
	Read(string, string) (models.KV, error)
	// RemoveUser Would delete all of the user info
	RemoveETHUser(string) error
	// RemoveUser Would delete all of the user info
	RemoveTMUser(string) error
	// Iterate over the entire database and find all the users
	Iterate() []models.User
	// IterateExpired would return a slice of expired users
	IterateExpired() ([]models.ExpiredUsers, error)
	// MultiRead would return a slice of your key value
	// pairs just to avoid too much redundant code
	MultiReader([]string, string) ([]models.KV, error)
	// MultiWriter would write multiple key value pairs
	// into database to avoid multiple calls to Insert() inside a method
	MultiWriter([]models.KV, string) error
}
