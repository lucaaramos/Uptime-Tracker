package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/lv99/uptime-tracker/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists = errors.New("username already exists")
	ErrNotFound   = errors.New("not found")
)

type AuthRepo struct {
	users   *mongo.Collection
	apiKeys *mongo.Collection
}

func NewAuthRepo(db *mongo.Database) *AuthRepo {
	return &AuthRepo{
		users:   db.Collection("users"),
		apiKeys: db.Collection("api_keys"),
	}
}

func (r *AuthRepo) EnsureIndexes(ctx context.Context) error {
	_, err := r.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create users username index: %w", err)
	}

	_, err = r.apiKeys.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key_hash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		return fmt.Errorf("create api_keys indexes: %w", err)
	}

	return nil
}

func HashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

func (r *AuthRepo) CreateUser(ctx context.Context, user *model.User) error {
	user.ID = primitive.NewObjectID()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	_, err := r.users.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrUserExists
		}
		return err
	}
	return nil
}

func (r *AuthRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.users.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepo) EnsureBootstrapAdmin(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return nil
	}
	_, err := r.GetUserByUsername(ctx, username)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}

	err = r.CreateUser(ctx, &model.User{
		Username:     username,
		PasswordHash: string(hash),
	})
	if errors.Is(err, ErrUserExists) {
		return nil
	}
	return err
}

func (r *AuthRepo) CreateAPIKey(ctx context.Context, key *model.APIKey) error {
	key.ID = primitive.NewObjectID()
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now()
	}
	_, err := r.apiKeys.InsertOne(ctx, key)
	return err
}

func (r *AuthRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var key model.APIKey
	err := r.apiKeys.FindOne(ctx, bson.M{"key_hash": keyHash}).Decode(&key)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (r *AuthRepo) ListAPIKeysByUser(ctx context.Context, userID primitive.ObjectID) ([]model.APIKey, error) {
	cursor, err := r.apiKeys.Find(
		ctx,
		bson.M{"user_id": userID},
		options.Find().SetSort(bson.M{"created_at": -1}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var keys []model.APIKey
	if err := cursor.All(ctx, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *AuthRepo) DeleteAPIKey(ctx context.Context, id, userID primitive.ObjectID) error {
	res, err := r.apiKeys.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AuthRepo) TouchAPIKeyLastUsed(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	_, err := r.apiKeys.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"last_used_at": now}},
	)
	return err
}
