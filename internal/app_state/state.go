package app_state

import (
	"github.com/Zallu35/blog-aggregator/internal/config"
	"github.com/Zallu35/blog-aggregator/internal/database"
)

type AppState struct {
	Config *config.Config
	DBConn *database.Queries
}
