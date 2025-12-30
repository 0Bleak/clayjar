package repository

import (
	"context"
	"time"

	"github.com/0Bleak/clayjar-jar-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type JarRepository interface {
	Create(ctx context.Context, jar *models.Jar) error
	FindByID(ctx context.Context, id string) (*models.Jar, error)
	FindAll(ctx context.Context, limit, offset int64) ([]*models.Jar, error)
	Update(ctx context.Context, id string, jar *models.Jar) error
	Delete(ctx context.Context, id string) error
	EnsureIndexes(ctx context.Context) error
}

type jarRepository struct {
	collection *mongo.Collection
}

func NewJarRepository(db *mongo.Database) JarRepository {
	return &jarRepository{
		collection: db.Collection("jars"),
	}
}

func (r *jarRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "category", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		{
			Keys: bson.D{{Key: "price", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "attributes.clay_type", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *jarRepository) Create(ctx context.Context, jar *models.Jar) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	jar.PrepareForCreate()

	_, err := r.collection.InsertOne(ctx, jar)
	return err
}

func (r *jarRepository) FindByID(ctx context.Context, id string) (*models.Jar, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var jar models.Jar
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&jar)
	if err != nil {
		return nil, err
	}

	return &jar, nil
}

func (r *jarRepository) FindAll(ctx context.Context, limit, offset int64) ([]*models.Jar, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	opts := options.Find().SetLimit(limit).SetSkip(offset).SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var jars []*models.Jar
	if err := cursor.All(ctx, &jars); err != nil {
		return nil, err
	}

	return jars, nil
}

func (r *jarRepository) Update(ctx context.Context, id string, jar *models.Jar) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	jar.PrepareForUpdate()

	update := bson.M{
		"$set": bson.M{
			"name":        jar.Name,
			"description": jar.Description,
			"category":    jar.Category,
			"price":       jar.Price,
			"stock_qty":   jar.StockQty,
			"image_url":   jar.ImageUrl,
			"attributes":  jar.Attributes,
			"updated_at":  jar.UpdatedAt,
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *jarRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}
