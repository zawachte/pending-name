package scheme

import (
	"github.com/acorn-io/baaah/pkg/merr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rancher/wrangler/pkg/schemes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func AddToScheme(scheme *runtime.Scheme) error {
	var errs []error
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	errs = append(errs, corev1.AddToScheme(scheme))
	//errs = append(errs, appsv1.AddToScheme(scheme))
	//errs = append(errs, policyv1.AddToScheme(scheme))
	//errs = append(errs, batchv1.AddToScheme(scheme))
	//errs = append(errs, networkingv1.AddToScheme(scheme))
	//errs = append(errs, storagev1.AddToScheme(scheme))
	//errs = append(errs, apiregistrationv1.AddToScheme(scheme))
	//errs = append(errs, rbacv1.AddToScheme(scheme))
	//errs = append(errs, authv1.AddToScheme(scheme))
	//errs = append(errs, apiextensionv1.AddToScheme(scheme))
	//errs = append(errs, discoveryv1.AddToScheme(scheme))
	//errs = append(errs, schedulingv1.AddToScheme(scheme))
	//errs = append(errs, coordinationv1.AddToScheme(scheme))
	return merr.NewErrors(errs...)
}

func init() {
	schemes.Register(AddToScheme)
	AddToScheme(Scheme)
}
