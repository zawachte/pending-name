package main

import (
	"context"
	"time"

	"github.com/acorn-io/mink/pkg/serializer"
	"github.com/acorn-io/mink/pkg/server"
	"github.com/pkg/errors"

	"github.com/acorn-io/mink/pkg/stores"
	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"

	"github.com/zawachte/pending-name/pkg/scheme"
	"github.com/zawachte/pending-name/pkg/services/pods"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	kube_scheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/generated/openapi"
)

var (
	opts = server.DefaultOpts()
)

func main() {
	err := idk()
	if err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Minute)
}

func idk() error {
	cfg, err := New(Config{
		Version:     "v0.0",
		DefaultOpts: opts,
	})
	if err != nil {
		return errors.Wrapf(err, "here 5")
	}

	if err := cfg.Run(context.Background()); err != nil {
		return errors.Wrapf(err, "here 6")
	}
	return nil
}

type Config struct {
	Version            string
	DefaultOpts        *options.RecommendedOptions
	IgnoreStartFailure bool
}

func New(cfg Config) (*server.Server, error) {

	dd, err := APIGroup()
	if err != nil {
		return nil, errors.Wrapf(err, "here 15")
	}
	return server.New(&server.Config{
		Name:                  "pending-name",
		Version:               cfg.Version,
		HTTPSListenPort:       7443,
		LongRunningVerbs:      []string{"watch", "proxy"},
		LongRunningResources:  []string{"exec", "proxy", "log", "registryport", "port", "push", "pull", "portforward", "copy"},
		OpenAPIConfig:         openapi.GetOpenAPIDefinitions,
		Scheme:                scheme.Scheme,
		CodecFactory:          &scheme.Codecs,
		APIGroups:             []*genericapiserver.APIGroupInfo{dd},
		DefaultOptions:        cfg.DefaultOpts,
		SupportAPIAggregation: false,
		IgnoreStartFailure:    cfg.IgnoreStartFailure,
	})
}

func NewClusterStorage() rest.Storage {
	//factory, err := db.NewFactory(scheme.Scheme, "mysql://zach:SisterLily1@@tcp(mysqlgroup.mysql.database.azure.com:3306)/test?parseTime=true")
	//if err != nil {
	//	panic(err)
	//}
	//strategy, err := factory.NewDBStrategy(&corev1.Pod{})
	//if err != nil {
	//	panic(err)
	//}

	svc, err := pods.NewPodService(scheme.Scheme, &corev1.Pod{})
	if err != nil {
		panic(err)
	}

	return stores.NewBuilder(scheme.Scheme, &corev1.Pod{}).
		WithCompleteCRUD(svc).
		WithTableConverter(rest.NewDefaultTableConvertor(corev1.Resource("pods"))).
		Build()
}

func Stores() (map[string]rest.Storage, error) {
	return map[string]rest.Storage{
		"pods": NewClusterStorage(),
	}, nil
}

func APIGroup() (*genericapiserver.APIGroupInfo, error) {

	stores, err := Stores()
	if err != nil {
		return nil, errors.Wrapf(err, "here2")
	}

	newScheme := runtime.NewScheme()
	err = kube_scheme.AddToScheme(newScheme)
	if err != nil {
		return nil, errors.Wrapf(err, "here")
	}

	gv := schema.GroupVersion{Group: "", Version: runtime.APIVersionInternal}

	err = v1.AddToSchemeWithGV(newScheme, gv)
	if err != nil {
		return nil, errors.Wrapf(err, "here1")
	}

	newScheme.AddKnownTypes(gv, &corev1.Pod{})

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo("", newScheme, scheme.ParameterCodec, scheme.Codecs)
	apiGroupInfo.VersionedResourcesStorageMap["v1"] = stores
	apiGroupInfo.NegotiatedSerializer = serializer.NewNoProtobufSerializer(apiGroupInfo.NegotiatedSerializer)

	return &apiGroupInfo, nil
}
