package distlock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/onexstack/onexstack/pkg/logger"
	gormlogger "gorm.io/gorm/logger"
)

// GORMLocker provides a distributed locking mechanism using GORM.
type GORMLocker struct {
	db          *gorm.DB
	lockName    string
	lockTimeout time.Duration
	renewTicker *time.Ticker
	stopChan    chan struct{}
	mu          sync.Mutex
	ownerID     string
	logger      logger.Logger
}

// Lock represents a database record for a distributed lock.
type Lock struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"unique"`
	OwnerID   string
	ExpiredAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Ensure GORMLocker implements the Locker interface.
var _ Locker = (*GORMLocker)(nil)

// NewGORMLocker initializes a new GORMLocker instance.
func NewGORMLocker(db *gorm.DB, opts ...Option) (*GORMLocker, error) {
	o := ApplyOptions(opts...)

	db = db.Session(&gorm.Session{Logger: gormlogger.Discard})

	if o.autoMigrate {
		if err := db.AutoMigrate(&Lock{}); err != nil {
			return nil, err
		}
	}

	locker := &GORMLocker{
		db:          db,
		ownerID:     o.ownerID,
		lockName:    o.lockName,
		lockTimeout: o.lockTimeout,
		stopChan:    make(chan struct{}),
		logger:      o.logger,
	}

	locker.logger.Info("GORMLocker initialized", "lockName", locker.lockName, "ownerID", locker.ownerID)

	return locker, nil
}

// Lock acquires the distributed lock.
//
// It takes over an expired lease with one atomic UPDATE before it tries to
// INSERT. The obvious shape — INSERT, and on a duplicate take the row over — is
// not usable on PostgreSQL: a unique violation aborts the surrounding
// transaction, so the recovery statements fail with "current transaction is
// aborted", and GORM also keeps the failed statement's error on the session.
// Updating first is correct under a race too: the WHERE clause is re-evaluated
// after the row lock is taken, so only one contender can see the lease as
// expired.
func (l *GORMLocker) Lock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	expiredAt := now.Add(l.lockTimeout)

	result := l.db.Model(&Lock{}).
		Where("name = ? AND expired_at < ?", l.lockName, now).
		Updates(map[string]any{
			"owner_id":   l.ownerID,
			"expired_at": expiredAt,
			"updated_at": now,
		})
	if result.Error != nil {
		l.logger.Error("failed to take over an expired lock", "error", result.Error)
		return result.Error
	}
	if result.RowsAffected > 0 {
		l.logger.Info("Lock expired, updated owner", "lockName", l.lockName, "newOwnerID", l.ownerID)
	}

	if result.RowsAffected == 0 {
		// Either no row exists, or the row belongs to a live owner. A duplicate
		// key means the latter (or a lease that expired between the two
		// statements, which the caller's retry loop handles).
		if err := l.db.Create(&Lock{Name: l.lockName, OwnerID: l.ownerID, ExpiredAt: expiredAt}).Error; err != nil {
			if isDuplicateEntry(err) {
				l.logger.Warn("lock is already held by another owner", "lockName", l.lockName)
				return fmt.Errorf("%w: %s", ErrLockHeld, l.lockName)
			}
			l.logger.Error("failed to create lock", "error", err)
			return err
		}
	}

	l.renewTicker = time.NewTicker(l.lockTimeout / 2)
	go l.renewLock(ctx)

	l.logger.Info("Lock acquired", "lockName", l.lockName, "ownerID", l.ownerID)
	return nil
}

// Unlock releases the distributed lock.
func (l *GORMLocker) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.renewTicker != nil {
		l.renewTicker.Stop()
		l.renewTicker = nil
		l.logger.Info("Stopped renewing lock", "lockName", l.lockName)
	}

	err := l.db.Delete(&Lock{}, "name = ?", l.lockName).Error
	if err != nil {
		l.logger.Error("failed to delete lock", "error", err)
		return err
	}

	l.logger.Info("Lock released", "lockName", l.lockName)
	return nil
}

// Renew refreshes the lease for the distributed lock.
func (l *GORMLocker) Renew(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	expiredAt := now.Add(l.lockTimeout)

	err := l.db.Model(&Lock{}).Where("name = ?", l.lockName).Update("expired_at", expiredAt).Error
	if err != nil {
		l.logger.Error("failed to renew lock", "error", err)
		return err
	}

	l.logger.Info("Lock renewed", "lockName", l.lockName, "newExpiration", expiredAt)
	return nil
}

// renewLock periodically renews the lock lease.
func (l *GORMLocker) renewLock(ctx context.Context) {
	for {
		select {
		case <-l.stopChan:
			return
		case <-l.renewTicker.C:
			if err := l.Renew(ctx); err != nil {
				l.logger.Error("failed to renew lock", "error", err)
			}
		}
	}
}

// isDuplicateEntry checks if the error is a duplicate entry error for MySQL and PostgreSQL.
func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}

	// GORM's postgres and mysql drivers translate a unique-constraint violation
	// into gorm.ErrDuplicatedKey before the caller sees it, so the
	// driver-specific checks below never match on those dialects. Without this
	// check an expired lock row cannot be taken over at all: every retry is a
	// second INSERT that comes back as ErrDuplicatedKey and is treated as a
	// genuine failure, so the lock is stuck forever.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		return mysqlErr.Number == 1062 // MySQL error code for duplicate entry
	}

	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505" // PostgreSQL error code for unique violation
	}

	/*
		if pgErr, ok := err.(sqlite3.Error); ok {
			return pgErr.Code == sqlite3.ErrConstraint
		}
	*/

	return false
}
