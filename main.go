package main

import (
	"fmt"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	_ "k8s.io/component-base/logs/json/register" // for JSON log format registration
	"k8s.io/kubernetes/cmd/kube-apiserver/app"
	"k8s.io/kubernetes/cmd/kube-apiserver/app/options"
)

func main() {
	err := idk()
	if err != nil {
		fmt.Print(fmt.Errorf(err.Error()))
	}
}

func idk() error {
	logs.InitLogs()

	s := options.NewServerRunOptions()
	// set default options
	completedOptions, err := app.Complete(s)
	if err != nil {
		return err
	}

	//if errs := completedOptions.Validate(); len(errs) != 0 {
	//	return utilerrors.NewAggregate(errs)
	//}

	completedOptions.Authentication.ServiceAccounts.Issuers = append(completedOptions.Authentication.ServiceAccounts.Issuers, "https://kubernetes.default.svc.cluster.local")
	completedOptions.Logs.Verbosity = 10

	server, err := app.CreateServerChain(completedOptions)
	if err != nil {
		return err
	}

	prepared, err := server.PrepareRun()
	if err != nil {
		return err
	}

	return prepared.Run(genericapiserver.SetupSignalHandler())
}
