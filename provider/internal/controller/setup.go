// Package controller wires Network and Instance into crossplane-runtime's
// managed resource reconciler.
package controller

import (
	"context"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/georgitinchev/gantry/provider/apis/v1alpha1"
)

type Options struct {
	Cloud        Cloud
	Logger       logging.Logger
	PollInterval time.Duration
}

func Setup(mgr ctrl.Manager, o Options) error {
	if err := setupNetwork(mgr, o); err != nil {
		return err
	}
	return setupInstance(mgr, o)
}

func reconcilerOptions(mgr ctrl.Manager, name string, o Options) []managed.ReconcilerOption {
	return []managed.ReconcilerOption{
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(o.PollInterval),
		// External names are cloud IDs assigned on create, never metadata.name.
		managed.WithInitializers(),
	}
}

func setupNetwork(mgr ctrl.Manager, o Options) error {
	name := "managed/network.cloud.gantry.internal"
	connector := managed.TypedExternalConnectorFn[*v1alpha1.Network](
		func(context.Context, *v1alpha1.Network) (managed.TypedExternalClient[*v1alpha1.Network], error) {
			return &networkExternal{cloud: o.Cloud}, nil
		})
	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.NetworkGroupVersionKind),
		append(reconcilerOptions(mgr, name, o), managed.WithTypedExternalConnector(connector))...)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		For(&v1alpha1.Network{}).
		Complete(r)
}

func setupInstance(mgr ctrl.Manager, o Options) error {
	name := "managed/instance.cloud.gantry.internal"
	kube := mgr.GetClient()
	connector := managed.TypedExternalConnectorFn[*v1alpha1.Instance](
		func(context.Context, *v1alpha1.Instance) (managed.TypedExternalClient[*v1alpha1.Instance], error) {
			return &instanceExternal{cloud: o.Cloud, kube: kube}, nil
		})
	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.InstanceGroupVersionKind),
		append(reconcilerOptions(mgr, name, o), managed.WithTypedExternalConnector(connector))...)
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		For(&v1alpha1.Instance{}).
		Complete(r)
}
