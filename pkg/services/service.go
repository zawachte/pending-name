package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/pkg/errors"
	"github.com/zawachte/pending-name/pkg/repositories"
	"github.com/zawachte/pending-name/pkg/services/pods"

	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/kubernetes/pkg/kubelet/kuberuntime"
)

var _ strategy.CompleteStrategy = (*Service)(nil)

type Service struct {
	obj              runtime.Object
	objList          runtime.Object
	scheme           *runtime.Scheme
	kubeletRuntime   kuberuntime.KubeGenericRuntime
	repo             repositories.Repository
	gvk              schema.GroupVersionKind
	componentService ComponentService
}

type ComponentService interface {
	Update(ctx context.Context, object types.Object) (types.Object, error)
	Create(ctx context.Context, object types.Object) (types.Object, error)
	Delete(ctx context.Context, object types.Object) error
}

func NewService(scheme *runtime.Scheme, obj runtime.Object) (*Service, error) {

	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, err
	}

	// test we can create objects
	_, err = scheme.New(gvk)
	if err != nil {
		return nil, err
	}

	objList, err := scheme.New(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	})

	repo, err := repositories.NewRepository()
	if err != nil {
		return nil, err
	}

	componentService, err := pods.NewService()
	if err != nil {
		return nil, err
	}

	return &Service{
		obj:              obj,
		objList:          objList,
		repo:             repo,
		gvk:              gvk,
		componentService: componentService,
	}, nil
}

func (r *Service) Create(ctx context.Context, object types.Object) (types.Object, error) {
	_, err := r.Get(ctx, object.GetNamespace(), object.GetName())
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		return nil, newAlreadyExists(r.gvk, object.GetName())
	}

	object.SetResourceVersion("zach")

	return r.put(ctx, object)
}

func (r *Service) put(ctx context.Context, object types.Object) (types.Object, error) {
	jsonBytes, err := json.Marshal(object)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to marshal create object to json")
	}

	key := keyFromObject(object)

	err = r.repo.Put(ctx, key, jsonBytes)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (r *Service) New() types.Object {
	fmt.Println("new")
	return r.obj.DeepCopyObject().(types.Object)
}

func newNotFound(gvk schema.GroupVersionKind, name string) error {
	return apierrors.NewNotFound(
		schema.GroupResource{
			Group:    gvk.Group,
			Resource: gvk.Kind,
		}, name)
}

func newAlreadyExists(gvk schema.GroupVersionKind, name string) error {
	return apierrors.NewAlreadyExists(
		schema.GroupResource{
			Group:    gvk.Group,
			Resource: gvk.Kind,
		}, name)
}

func (r *Service) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	fmt.Println("get")
	key := fmt.Sprintf("/registrys/%s/%s/%s",
		namespace,
		strings.ToLower(r.gvk.Kind),
		name)

	v, resourceVersion, err := r.repo.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, newNotFound(r.gvk, name)
	}

	obj := r.obj.DeepCopyObject()
	err = json.Unmarshal(v, obj)
	if err != nil {
		return nil, err
	}

	typesObject := obj.(types.Object)
	typesObject.SetResourceVersion(resourceVersion)

	return typesObject, nil
}

func (r *Service) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	fmt.Println("update")
	return r.put(ctx, obj)
}

func (r *Service) UpdateStatus(ctx context.Context, obj types.Object) (types.Object, error) {
	fmt.Println("updatestatus")
	return r.put(ctx, obj)
}

func (r *Service) GetToList(ctx context.Context, namespace, name string) (types.ObjectList, error) {
	obj, err := r.Get(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	list := r.objList.DeepCopyObject().(types.ObjectList)
	return list, meta.SetList(list, []runtime.Object{obj})
}

func (r *Service) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	prefix := fmt.Sprintf("/registrys/%s/%s", namespace, strings.ToLower(r.gvk.Kind))
	list := r.objList.DeepCopyObject().(types.ObjectList)

	values, err := r.repo.List(ctx, prefix, opts)
	if err != nil {
		return nil, err
	}

	objList := []runtime.Object{}
	for _, v := range values {
		obj := r.obj.DeepCopyObject()
		err = json.Unmarshal(v, obj)
		if err != nil {
			return nil, err
		}

		objList = append(objList, obj)
	}

	return list, meta.SetList(list, objList)
}

func (r *Service) NewList() types.ObjectList {
	return r.objList.DeepCopyObject().(types.ObjectList)
}

func keyFromObject(obj types.Object) string {
	return fmt.Sprintf("/registrys/%s/%s/%s",
		obj.GetNamespace(),
		strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind),
		obj.GetName())
}

func (r *Service) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	key := keyFromObject(obj)

	err := r.repo.Delete(ctx, key)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (r *Service) Watch(ctx context.Context, namespace string, opts storage.ListOptions) (<-chan watch.Event, error) {
	return nil, nil
}

func (r *Service) Destroy() {
}

func (r *Service) Scheme() *runtime.Scheme {
	fmt.Println("scheme")
	return r.scheme
}
