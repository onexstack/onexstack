// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/onexstack/onexstack/pkg/store/logger/empty"
	"github.com/onexstack/onexstack/pkg/store/where"
)

// DBProvider defines an interface for providing a database connection.
type DBProvider interface {
	// DB returns the database instance for the given context.
	DB(ctx context.Context, wheres ...where.Where) *gorm.DB
}

// Option defines a function type for configuring the Store.
type Option[T any] func(*Store[T])

// Store represents a generic data store with logging capabilities.
type Store[T any] struct {
	logger  Logger
	storage DBProvider
}

// WithLogger returns an Option function that sets the provided Logger to the Store for logging purposes.
func WithLogger[T any](logger Logger) Option[T] {
	return func(s *Store[T]) {
		s.logger = logger
	}
}

// NewStore creates a new instance of Store with the provided DBProvider.
func NewStore[T any](storage DBProvider, logger Logger) *Store[T] {
	if logger == nil {
		logger = empty.NewLogger()
	}

	return &Store[T]{
		logger:  logger,
		storage: storage,
	}
}

// db retrieves the database instance and applies the provided where conditions.
//
// The context is bound here rather than trusted to the provider: a DB that reads
// the transaction out of ctx but does not call WithContext returns a handle that
// carries the request's transaction but not its cancellation or deadline.
func (s *Store[T]) db(ctx context.Context, wheres ...where.Where) *gorm.DB {
	dbInstance := s.storage.DB(ctx).WithContext(ctx)
	for _, whr := range wheres {
		if whr != nil {
			dbInstance = whr.Where(dbInstance)
		}
	}
	return dbInstance
}

// Create inserts a new object into the database.
func (s *Store[T]) Create(ctx context.Context, obj *T) error {
	if err := s.db(ctx).Create(obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to insert object into database", "object", obj)
		return err
	}
	return nil
}

// Update modifies an existing object in the database.
func (s *Store[T]) Update(ctx context.Context, obj *T) error {
	if err := s.db(ctx).Save(obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to update object in database", "object", obj)
		return err
	}
	return nil
}

// Delete removes an object from the database based on the provided where options.
func (s *Store[T]) Delete(ctx context.Context, opts *where.Options) error {
	err := s.db(ctx, opts).Delete(new(T)).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// s.logger.Error(ctx, err, "Failed to delete object from database")
		return err
	}
	return nil
}

// Get retrieves a single object from the database based on the provided where options.
func (s *Store[T]) Get(ctx context.Context, opts *where.Options) (*T, error) {
	var obj T
	if err := s.db(ctx, opts).First(&obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to retrieve object from database")
		return nil, err
	}
	return &obj, nil
}

// List retrieves a list of objects from the database based on the provided where options.
//
// It returns the page and the total it is drawn from, which is two statements:
// a COUNT over the whole filtered set and a paged SELECT over it. The count
// comes first, so a filter that matches nothing costs one statement instead of
// two — an empty page is not rare (a user with no orders, a tag with no posts),
// and the SELECT it skips is the one that has to build and sort a result set.
//
// # The two statements are forked, not chained
//
// Session(&gorm.Session{}) gives each statement its own copy of the built query.
// Chaining them — Find followed by Offset(-1).Limit(-1).Count — happens to work
// because Count rebuilds its own SQL, but it leaves the two statements sharing
// one *gorm.Statement, and the paging reset is a side effect either of them
// could observe. The fork states the intent: the count is over the whole set
// (offset -1 and limit -1 are GORM's "no limit"), the page is over its own
// slice of it.
//
// opts must not be reused across calls: Options.Where appends to the receiver's
// own clause slice, so a second call re-applies every condition.
func (s *Store[T]) List(ctx context.Context, opts *where.Options) (count int64, ret []*T, err error) {
	if opts == nil {
		opts = where.NewWhere()
	}
	// Model is set from T rather than left to be inferred from a destination:
	// the count runs first and carries no destination of its own, so without it
	// GORM has no table to count — "Table not set, please set it like:
	// db.Model(&user)". Find would have inferred it, but it is the statement that
	// runs second.
	query := opts.Where(s.db(ctx).Model(new(T)))

	if err = query.Session(&gorm.Session{}).Offset(-1).Limit(-1).Count(&count).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to list objects from database")
		return 0, nil, err
	}
	if count == 0 {
		return 0, nil, nil
	}

	if err = query.Order("id desc").Find(&ret).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to list objects from database")
		return 0, nil, err
	}
	return count, ret, nil
}
