package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Actives struct {
	Symbol     string     `json:"symbol" bson:"symbol"`
	Amount     string     `json:"amount" bson:"amount"`
	Side       string     `json:"side" bson:"side"`
	Round      int        `json:"round" bson:"round"`
	Price      string     `json:"price" bson:"price"`
	HighestROE *float64   `json:"highest_roe" bson:"highest_roe"`
	UpdatedAt  *time.Time `json:"updated_at" bson:"updated_at"`
}

type Logs struct {
	ClientOrderID string    `json:"client_order_id" bson:"client_order_id"`
	OrderID       int64     `json:"order_id" bson:"order_id"`
	Symbol        string    `json:"symbol" bson:"symbol"`
	Amount        string    `json:"amount" bson:"amount"`
	Leverage      string    `json:"leverage" bson:"leverage"`
	Price         string    `json:"price" bson:"price"`
	Side          string    `json:"side" bson:"side"`
	Position      string    `json:"position" bson:"position"`
	Round         int       `json:"round" bson:"round"`
	Profilt       string    `json:"profit" bson:"profit"`
	ROE           float64   `json:"roe" bson:"roe"`
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`
}

type IMongoRepository interface {
	FindAndUpdateAction(active Actives) error
	FindOneActivesWithSymbol(symbol string) (*Actives, error)
	CreateLog(logs Logs) error
}

type MongoRepository struct {
	mongoDB         *Mongo
	mongoActivesCol *mongodriver.Collection
	mongoLogsCol    *mongodriver.Collection
}

func NewMongoRepository(mongoDB *Mongo) IMongoRepository {
	mongoActivesCol := mongoDB.Client.Database("sigbot").Collection("actives")
	mongoLogsCol := mongoDB.Client.Database("sigbot").Collection("logs")
	return &MongoRepository{
		mongoDB:         mongoDB,
		mongoActivesCol: mongoActivesCol,
		mongoLogsCol:    mongoLogsCol,
	}
}

func (repo *MongoRepository) FindAndUpdateAction(active Actives) error {
	ctx := context.Background()
	db := repo.mongoActivesCol
	now := time.Now()
	isUpsert := true
	active.UpdatedAt = &now
	filter := bson.M{"symbol": active.Symbol}
	update := bson.M{
		"$set": bson.M{
			"symbol":      active.Symbol,
			"amount":      active.Amount,
			"side":        active.Side,
			"round":       active.Round,
			"price":       active.Price,
			"highest_roe": active.HighestROE,
			"updated_at":  active.UpdatedAt,
		},
	}
	res := db.FindOneAndUpdate(ctx, filter, update, &options.FindOneAndUpdateOptions{Upsert: &isUpsert})
	return res.Err()
}

func (repo *MongoRepository) FindOneActivesWithSymbol(symbol string) (*Actives, error) {
	ctx := context.Background()
	db := repo.mongoActivesCol
	filter := bson.M{"symbol": symbol}
	act := &Actives{}
	err := db.FindOne(ctx, filter).Decode(act)
	if err != nil {
		return nil, err
	}

	return act, nil
}

func (repo *MongoRepository) CreateLog(logs Logs) error {
	ctx := context.Background()
	db := repo.mongoLogsCol
	_, err := db.InsertOne(ctx, logs)
	if err != nil {
		return err
	}

	return nil
}
