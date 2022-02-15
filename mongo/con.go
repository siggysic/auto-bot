package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	ctx context.Context
	uri string

	Client *mongo.Client
}

func NewMongo(ctx context.Context, uri string) *Mongo {
	return &Mongo{
		ctx: ctx,
		uri: uri,
	}
}

func (m *Mongo) Connect() error {
	ctx := m.ctx
	uri := m.uri
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	m.Client = client
	return nil
}

func (m *Mongo) Disconnect() error {
	ctx := m.ctx
	client := m.Client

	return client.Disconnect(ctx)
}
