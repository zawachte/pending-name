package pods

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/acorn-io/mink/pkg/types"
	"github.com/zawachte/pending-name/pkg/kubeletruntime"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/kuberuntime"
)

type service struct {
	kubeletRuntime kuberuntime.KubeGenericRuntime
}

const (
	retryPeriod    = 1 * time.Second
	maxRetryPeriod = 20 * time.Second
)

func (s *service) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	fmt.Println("create pod service")

	pod := obj.(*v1.Pod)
	backoff := flowcontrol.NewBackOff(retryPeriod, maxRetryPeriod)
	s.kubeletRuntime.SyncPod(ctx, pod, &container.PodStatus{}, []corev1.Secret{}, backoff)

	return obj, nil
}

func (s *service) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	fmt.Println("update pod service")
	backoff := flowcontrol.NewBackOff(retryPeriod, maxRetryPeriod)
	pod := obj.(*v1.Pod)
	s.kubeletRuntime.SyncPod(ctx, pod, &container.PodStatus{}, []corev1.Secret{}, backoff)
	return obj, nil
}

func (s *service) Delete(ctx context.Context, obj types.Object) error {
	fmt.Println("delete pod service")
	//pod := obj.(*v1.Pod)

	//s.kubeletRuntime.KillPod(ctx, pod)
	return nil
}

func NewService() (*service, error) {

	rt, err := kubeletruntime.NewKubeletRuntime()
	if err != nil {
		return nil, err
	}

	return &service{
		kubeletRuntime: rt,
	}, nil
}
