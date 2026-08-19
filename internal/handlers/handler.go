package handlers

import (
    "github.com/betking/rich-backend/config"
    "github.com/betking/rich-backend/internal/database"
    "github.com/betking/rich-backend/internal/engine"
    ws "github.com/betking/rich-backend/internal/websocket"
)

type Handler struct {
    DB     *database.PostgresDB
    Redis  *database.RedisDB
    Mongo  *database.MongoDB
    Config *config.Config
    Engine *engine.LeagueProcessor
    WSHub  *ws.Hub
}

func NewHandler(pg *database.PostgresDB, redis *database.RedisDB, mongo *database.MongoDB, cfg *config.Config) *Handler {
    return &Handler{
        DB:     pg,
        Redis:  redis,
        Mongo:  mongo,
        Config: cfg,
        WSHub:  ws.NewHub(),
    }
}