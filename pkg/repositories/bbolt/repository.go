package bbolt

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"go.etcd.io/bbolt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
)

type repository struct {
	db *bbolt.DB
}

func NewRepository() (*repository, error) {
	db, err := bbolt.Open("my.db", 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("global-bucket"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &repository{
		db: db,
	}, nil
}

func (r *repository) Close() {
	r.db.Close()
}

func (r *repository) Put(ctx context.Context, key string, value []byte) error {
	err := r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("global-bucket"))

		resourceVersion, err := b.NextSequence()
		if err != nil {
			return err
		}

		resourceVersionString := strconv.FormatUint(resourceVersion, 10)
		entryKey := []byte(fmt.Sprintf("%s+%s", key, resourceVersionString))
		err = b.Put(entryKey, value)
		return err
	})

	return err
}

func (r *repository) List(ctx context.Context, prefix string, opts storage.ListOptions) ([][]byte, error) {
	values := [][]byte{}
	prefixBytes := []byte(prefix)

	r.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket([]byte("global-bucket")).Cursor()
		for k, v := c.Seek(prefixBytes); k != nil && bytes.HasPrefix(k, prefixBytes); k, v = c.Next() {
			values = append(values, v)
		}
		return nil
	})

	return values, nil
}

func (r *repository) Get(ctx context.Context, key string) ([]byte, string, error) {
	value := []byte{}
	resourceVersion := ""
	prefixBytes := []byte(key)

	err := r.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket([]byte("global-bucket")).Cursor()
		for k, v := c.Seek(prefixBytes); k != nil && bytes.HasPrefix(k, prefixBytes); k, v = c.Next() {
			delimbedString := strings.Split(string(key), "+")
			resourceVersion = delimbedString[len(delimbedString)-1]
			value = v
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	return value, resourceVersion, nil
}

func newNotFound(gvk schema.GroupVersionKind, name string) error {
	return apierrors.NewNotFound(
		schema.GroupResource{
			Group:    gvk.Group,
			Resource: gvk.Kind,
		}, name)
}

func (r *repository) Delete(ctx context.Context, key string) error {
	err := r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("global-bucket"))
		err := b.Delete([]byte(key))
		return err
	})

	return err
}
