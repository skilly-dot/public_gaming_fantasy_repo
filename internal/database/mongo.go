// internal/database/mongo.go
package database

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
    Client   *mongo.Client
    Database *mongo.Database
}

func NewMongo(uri string) (*MongoDB, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }

    if err := client.Ping(ctx, nil); err != nil {
        return nil, err
    }

    return &MongoDB{
        Client:   client,
        Database: client.Database("betking_rich"),
    }, nil
}

func (m *MongoDB) Close() {
    m.Client.Disconnect(context.Background())
}

func (m *MongoDB) PlayersCol() *mongo.Collection {
    return m.Database.Collection("players")
}

func (m *MongoDB) CoachesCol() *mongo.Collection {
    return m.Database.Collection("coaches")
}

func (m *MongoDB) TeamsCol() *mongo.Collection {
    return m.Database.Collection("teams")
}