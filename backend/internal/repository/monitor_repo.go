package repository

import (
	"context"
	"time"

	"github.com/lv99/uptime-tracker/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MonitorRepo struct {
	coll *mongo.Collection
}

func NewMonitorRepo(db *mongo.Database) *MonitorRepo {
	return &MonitorRepo{coll: db.Collection("monitors")}
}

func (r *MonitorRepo) Create(ctx context.Context, m *model.Monitor) error {
	m.ID = primitive.NewObjectID()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	if m.Status == "" {
		m.Status = "active"
	}
	if m.Method == "" {
		m.Method = "GET"
	}
	if m.Timeout == 0 {
		m.Timeout = 5000
	}
	if m.Interval == 0 {
		m.Interval = 60
	}
	_, err := r.coll.InsertOne(ctx, m)
	return err
}

func (r *MonitorRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Monitor, error) {
	var m model.Monitor
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MonitorRepo) List(ctx context.Context) ([]model.Monitor, error) {
	cursor, err := r.coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var monitors []model.Monitor
	if err := cursor.All(ctx, &monitors); err != nil {
		return nil, err
	}
	return monitors, nil
}

func (r *MonitorRepo) ListActive(ctx context.Context) ([]model.Monitor, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"status": "active"}, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var monitors []model.Monitor
	if err := cursor.All(ctx, &monitors); err != nil {
		return nil, err
	}
	return monitors, nil
}

func (r *MonitorRepo) Update(ctx context.Context, id primitive.ObjectID, req model.MonitorRequest) error {
	update := bson.M{
		"$set": bson.M{
			"name":          req.Name,
			"url":           req.URL,
			"method":        req.Method,
			"interval_s":    req.Interval,
			"timeout_ms":    req.Timeout,
			"alert_email":   req.AlertEmail,
			"alert_webhook": req.AlertWebhook,
			"updated_at":    time.Now(),
		},
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *MonitorRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *MonitorRepo) SetStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{"status": status, "updated_at": time.Now()},
	})
	return err
}
