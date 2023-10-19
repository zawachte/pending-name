package kubeletruntime

import (
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/kubernetes/cmd/kubelet/app"
	kubeletoptions "k8s.io/kubernetes/cmd/kubelet/app/options"
	"k8s.io/kubernetes/pkg/kubelet"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	"k8s.io/kubernetes/pkg/kubelet/cm"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/kuberuntime"
	"k8s.io/kubernetes/pkg/kubelet/logs"
	proberesults "k8s.io/kubernetes/pkg/kubelet/prober/results"
)

func NewKubeletRuntime() (kuberuntime.KubeGenericRuntime, error) {

	maxPulls := int32(10)

	kubeletFlags := kubeletoptions.NewKubeletFlags()
	kubeletConfig, err := kubeletoptions.NewKubeletConfiguration()
	if err != nil {
		return nil, err
	}

	// construct a KubeletServer from kubeletFlags and kubeletConfig
	kubeletServer := &kubeletoptions.KubeletServer{
		KubeletFlags:         *kubeletFlags,
		KubeletConfiguration: *kubeletConfig,
	}

	kubeletDeps, err := app.UnsecuredDependencies(kubeletServer, utilfeature.DefaultFeatureGate)
	if err != nil {
		return nil, err
	}

	imageFsInfoProvider := cadvisor.NewImageFsInfoProvider(kubeletServer.ContainerRuntimeEndpoint)
	kubeletDeps.CAdvisorInterface, err = cadvisor.New(imageFsInfoProvider, kubeletServer.RootDirectory, []string{}, cadvisor.UsingLegacyCadvisorStats(kubeletServer.ContainerRuntimeEndpoint), kubeletServer.LocalStorageCapacityIsolation)
	if err != nil {
		return nil, err
	}

	machineInfo, err := kubeletDeps.CAdvisorInterface.MachineInfo()
	if err != nil {
		return nil, err
	}

	// setup containerLogManager for CRI container runtime
	containerLogManager, err := logs.NewContainerLogManager(
		kubeletDeps.RemoteRuntimeService,
		kubeletDeps.OSInterface,
		kubeletServer.ContainerLogMaxSize,
		int(kubeletServer.ContainerLogMaxFiles),
	)
	if err != nil {
		return nil, err
	}

	imageBackOff := flowcontrol.NewBackOff(time.Second*10, 300*time.Second)

	if err := kubelet.PreInitRuntimeService(&kubeletServer.KubeletConfiguration, kubeletDeps); err != nil {
		return nil, err
	}

	kubeletDeps.ContainerManager, err = cm.NewContainerManager(
		kubeletDeps.Mounter,
		kubeletDeps.CAdvisorInterface,
		cm.NodeConfig{
			RuntimeCgroupsName:    kubeletServer.RuntimeCgroups,
			SystemCgroupsName:     kubeletServer.SystemCgroups,
			KubeletCgroupsName:    kubeletServer.KubeletCgroups,
			KubeletOOMScoreAdj:    kubeletServer.OOMScoreAdj,
			CgroupsPerQOS:         false, //kubeletServer.CgroupsPerQOS,
			CgroupRoot:            kubeletServer.CgroupRoot,
			CgroupDriver:          kubeletServer.CgroupDriver,
			KubeletRootDir:        kubeletServer.RootDirectory,
			ProtectKernelDefaults: kubeletServer.ProtectKernelDefaults,
			NodeAllocatableConfig: cm.NodeAllocatableConfig{
				KubeReservedCgroupName:   kubeletServer.KubeReservedCgroup,
				SystemReservedCgroupName: kubeletServer.SystemReservedCgroup,
				EnforceNodeAllocatable:   sets.NewString(kubeletServer.EnforceNodeAllocatable...),
				KubeReserved:             nil,
				SystemReserved:           nil,
				ReservedSystemCPUs:       cpuset.New(),
				HardEvictionThresholds:   nil,
			},
			QOSReserved:                             nil,
			CPUManagerPolicy:                        kubeletServer.CPUManagerPolicy,
			CPUManagerPolicyOptions:                 map[string]string{},
			CPUManagerReconcilePeriod:               kubeletServer.CPUManagerReconcilePeriod.Duration,
			ExperimentalMemoryManagerPolicy:         kubeletServer.MemoryManagerPolicy,
			ExperimentalMemoryManagerReservedMemory: kubeletServer.ReservedMemory,
			PodPidsLimit:                            kubeletServer.PodPidsLimit,
			EnforceCPULimits:                        kubeletServer.CPUCFSQuota,
			CPUCFSQuotaPeriod:                       kubeletServer.CPUCFSQuotaPeriod.Duration,
			TopologyManagerPolicy:                   kubeletServer.TopologyManagerPolicy,
			TopologyManagerScope:                    kubeletServer.TopologyManagerScope,
			//TopologyManagerPolicyOptions:            map[string]string{},
		},
		false,
		kubeletDeps.Recorder,
		kubeletDeps.KubeClient,
	)
	if err != nil {
		return nil, err
	}

	runtime, err := kuberuntime.NewKubeGenericRuntimeManager(
		kubecontainer.FilterEventRecorder(kubeletDeps.Recorder),
		proberesults.NewManager(),
		proberesults.NewManager(),
		proberesults.NewManager(),
		kubeletServer.RootDirectory,
		machineInfo,
		nil, //klet.podWorkers,
		kubeletDeps.OSInterface,
		nil, //klet,
		&http.Client{},
		imageBackOff,
		true, //kubeCfg.SerializeImagePulls,
		&maxPulls,
		float32(5.0),
		int(10),
		kubeletServer.ImageCredentialProviderConfigFile,
		kubeletServer.ImageCredentialProviderBinDir,
		kubeletServer.CPUCFSQuota,
		kubeletServer.CPUCFSQuotaPeriod,
		kubeletDeps.RemoteRuntimeService,
		kubeletDeps.RemoteImageService,
		kubeletDeps.ContainerManager,
		containerLogManager,
		nil,
		kubeletServer.KubeletFlags.SeccompDefault || kubeletServer.KubeletConfiguration.SeccompDefault,
		kubeletServer.MemorySwap.SwapBehavior,
		kubeletDeps.ContainerManager.GetNodeAllocatableAbsolute,
		*kubeletServer.MemoryThrottlingFactor,
		kubeletDeps.PodStartupLatencyTracker,
		kubeletDeps.TracerProvider,
	)
	if err != nil {
		return nil, err
	}

	return runtime, nil
}
