package repositories

import (
	"context"

	"github.com/zawachte/pending-name/pkg/repositories/bbolt"

	"k8s.io/apiserver/pkg/storage"
)

type Repository interface {
	Put(ctx context.Context, key string, value []byte) error
	List(ctx context.Context, prefix string, opts storage.ListOptions) ([][]byte, error)
	Get(ctx context.Context, key string) ([]byte, string, error)
	Delete(ctx context.Context, key string) error
}

func NewRepository() (Repository, error) {
	return bbolt.NewRepository()
}
