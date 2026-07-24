package repository

import (
	"context"
	"sort"
	"time"

	"github.com/lv99/uptime-tracker/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CheckRepo struct {
	coll *mongo.Collection
}

func NewCheckRepo(db *mongo.Database) *CheckRepo {
	return &CheckRepo{coll: db.Collection("check_results")}
}

func (r *CheckRepo) Insert(ctx context.Context, cr *model.CheckResult) error {
	cr.ID = primitive.NewObjectID()
	if cr.CheckedAt.IsZero() {
		cr.CheckedAt = time.Now()
	}
	_, err := r.coll.InsertOne(ctx, cr)
	return err
}

func (r *CheckRepo) GetByMonitorID(ctx context.Context, monitorID primitive.ObjectID, limit, offset int) ([]model.CheckResult, error) {
	filter := bson.M{"monitor_id": monitorID}
	opts := options.Find().
		SetSort(bson.M{"checked_at": -1}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []model.CheckResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *CheckRepo) GetLatestByMonitorID(ctx context.Context, monitorID primitive.ObjectID) (*model.CheckResult, error) {
	var cr model.CheckResult
	opts := options.FindOne().SetSort(bson.M{"checked_at": -1})
	err := r.coll.FindOne(ctx, bson.M{"monitor_id": monitorID}, opts).Decode(&cr)
	if err != nil {
		return nil, err
	}
	return &cr, nil
}

func (r *CheckRepo) GetStats(ctx context.Context, monitorID primitive.ObjectID, since time.Time) (*model.Stats, error) {
	match := bson.M{
		"monitor_id": monitorID,
		"checked_at": bson.M{"$gte": since},
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "success", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: []interface{}{"$success", 1, 0}}}}}},
			{Key: "failures", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: []interface{}{"$success", 0, 1}}}}}},
			{Key: "avg_latency", Value: bson.D{{Key: "$avg", Value: "$latency_ms"}}},
		}}},
	}

	stats := &model.Stats{}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result struct {
			Total      int     `bson:"total"`
			Success    int     `bson:"success"`
			Failures   int     `bson:"failures"`
			AvgLatency float64 `bson:"avg_latency"`
		}
		if err := cursor.Decode(&result); err == nil {
			stats.TotalChecks = result.Total
			stats.SuccessChecks = result.Success
			stats.FailureChecks = result.Failures
			stats.AvgLatencyMs = result.AvgLatency
			if result.Total > 0 {
				stats.UptimePercent = float64(result.Success) / float64(result.Total) * 100
			}
		}
	}

	// Calculate percentiles
	latencies, err := r.getLatencies(ctx, monitorID, since)
	if err == nil && len(latencies) > 0 {
		sort.Float64s(latencies)
		stats.P95LatencyMs = percentile(latencies, 95)
		stats.P99LatencyMs = percentile(latencies, 99)
	}

	return stats, nil
}

func (r *CheckRepo) getLatencies(ctx context.Context, monitorID primitive.ObjectID, since time.Time) ([]float64, error) {
	cursor, err := r.coll.Find(ctx, bson.M{
		"monitor_id": monitorID,
		"success":    true,
		"checked_at": bson.M{"$gte": since},
	}, options.Find().SetProjection(bson.M{"latency_ms": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var latencies []float64
	for cursor.Next(ctx) {
		var r struct {
			Latency float64 `bson:"latency_ms"`
		}
		if err := cursor.Decode(&r); err == nil {
			latencies = append(latencies, r.Latency)
		}
	}
	return latencies, nil
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted) / 100)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
