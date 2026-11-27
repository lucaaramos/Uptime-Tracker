package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lv99/uptime-tracker/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IncidentRepo struct {
	coll *mongo.Collection
}

func NewIncidentRepo(db *mongo.Database) *IncidentRepo {
	return &IncidentRepo{coll: db.Collection("incidents")}
}

func (r *IncidentRepo) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "monitor_id", Value: 1},
				{Key: "started_at", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "started_at", Value: -1},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("create incidents indexes: %w", err)
	}
	return nil
}

// OpenOrUpdate opens a new incident if none is currently open for monitorID,
// or updates the existing open incident's fail count and latest cause.
func (r *IncidentRepo) OpenOrUpdate(
	ctx context.Context,
	monitorID primitive.ObjectID,
	monitorName, url, cause string,
	statusCode, failCount int,
	at time.Time,
) (*model.Incident, bool, error) {
	if at.IsZero() {
		at = time.Now()
	}

	var existing model.Incident
	err := r.coll.FindOne(
		ctx,
		bson.M{"monitor_id": monitorID, "status": model.IncidentStatusOpen},
		options.FindOne().SetSort(bson.M{"started_at": -1}),
	).Decode(&existing)

	if err == nil {
		update := bson.M{
			"$set": bson.M{
				"fail_count":  failCount,
				"cause":       cause,
				"status_code": statusCode,
			},
		}
		if _, uerr := r.coll.UpdateOne(ctx, bson.M{"_id": existing.ID}, update); uerr != nil {
			return nil, false, uerr
		}
		existing.FailCount = failCount
		existing.Cause = cause
		existing.StatusCode = statusCode
		return &existing, false, nil
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false, err
	}

	inc := &model.Incident{
		ID:          primitive.NewObjectID(),
		MonitorID:   monitorID,
		MonitorName: monitorName,
		URL:         url,
		Status:      model.IncidentStatusOpen,
		Cause:       cause,
		StatusCode:  statusCode,
		FailCount:   failCount,
		StartedAt:   at,
	}
	if _, err := r.coll.InsertOne(ctx, inc); err != nil {
		return nil, false, err
	}
	return inc, true, nil
}

// ResolveOpen resolves the active open incident for monitorID, if any.
// Returns (nil, nil) if no open incident exists.
func (r *IncidentRepo) ResolveOpen(ctx context.Context, monitorID primitive.ObjectID, at time.Time) (*model.Incident, error) {
	if at.IsZero() {
		at = time.Now()
	}

	var open model.Incident
	err := r.coll.FindOne(
		ctx,
		bson.M{"monitor_id": monitorID, "status": model.IncidentStatusOpen},
		options.FindOne().SetSort(bson.M{"started_at": -1}),
	).Decode(&open)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	durationSec := int64(at.Sub(open.StartedAt).Seconds())
	if durationSec < 0 {
		durationSec = 0
	}

	_, err = r.coll.UpdateOne(
		ctx,
		bson.M{"_id": open.ID},
		bson.M{
			"$set": bson.M{
				"status":      model.IncidentStatusResolved,
				"resolved_at": at,
				"duration_s":  durationSec,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	open.Status = model.IncidentStatusResolved
	open.ResolvedAt = &at
	open.DurationSec = durationSec
	return &open, nil
}

func (r *IncidentRepo) ListByMonitor(ctx context.Context, monitorID primitive.ObjectID, limit, offset int) ([]model.Incident, error) {
	opts := options.Find().
		SetSort(bson.M{"started_at": -1}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.coll.Find(ctx, bson.M{"monitor_id": monitorID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var incidents []model.Incident
	if err := cursor.All(ctx, &incidents); err != nil {
		return nil, err
	}
	return incidents, nil
}

func (r *IncidentRepo) ListRecent(ctx context.Context, status string, limit, offset int) ([]model.Incident, error) {
	filter := bson.M{}
	if status == model.IncidentStatusOpen || status == model.IncidentStatusResolved {
		filter["status"] = status
	}

	opts := options.Find().
		SetSort(bson.M{"started_at": -1}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var incidents []model.Incident
	if err := cursor.All(ctx, &incidents); err != nil {
		return nil, err
	}
	return incidents, nil
}
