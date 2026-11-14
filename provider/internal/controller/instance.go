package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/georgitinchev/gantry/provider/apis/v1alpha1"
	"github.com/georgitinchev/gantry/provider/internal/clients"
)

var errNetworkNotReady = errors.New("referenced network has no cloud ID yet")

type instanceExternal struct {
	cloud Cloud
	kube  client.Reader
}

func (e *instanceExternal) Observe(ctx context.Context, cr *v1alpha1.Instance) (managed.ExternalObservation, error) {
	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	i, err := e.cloud.GetInstance(ctx, id)
	if clients.IsNotFound(err) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("observe instance: %w", err)
	}
	cr.Status.AtProvider = v1alpha1.InstanceObservation{ID: i.ID, IP: i.IP, State: i.State, NetworkID: i.NetworkID, Size: i.Size}
	switch i.State {
	case "running":
		cr.SetConditions(xpv2.Available())
	case "provisioning":
		cr.SetConditions(xpv2.Creating())
	default:
		cr.SetConditions(xpv2.Unavailable().WithMessage("instance is " + i.State))
	}
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: i.Size == cr.Spec.ForProvider.Size,
		ConnectionDetails: managed.ConnectionDetails{
			"ip": []byte(i.IP),
		},
	}, nil
}

func (e *instanceExternal) networkID(ctx context.Context, cr *v1alpha1.Instance) (string, error) {
	net := &v1alpha1.Network{}
	key := types.NamespacedName{Namespace: cr.Namespace, Name: cr.Spec.ForProvider.NetworkRef.Name}
	if err := e.kube.Get(ctx, key, net); err != nil {
		return "", fmt.Errorf("get network %s: %w", key, err)
	}
	if net.Status.AtProvider.ID == "" {
		return "", fmt.Errorf("%w: %s", errNetworkNotReady, key)
	}
	return net.Status.AtProvider.ID, nil
}

func (e *instanceExternal) Create(ctx context.Context, cr *v1alpha1.Instance) (managed.ExternalCreation, error) {
	netID, err := e.networkID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	cr.SetConditions(xpv2.Creating())
	i, err := e.cloud.CreateInstance(ctx, cloudName(cr.Namespace, cr.Name), netID, cr.Spec.ForProvider.Size)
	if apiErr, conflict := clients.IsConflict(err); conflict {
		existing := &clients.Instance{}
		if jerr := json.Unmarshal(apiErr.Existing, existing); jerr != nil || existing.ID == "" {
			return managed.ExternalCreation{}, fmt.Errorf("create instance: conflict without existing object: %w", err)
		}
		i, err = existing, nil
	}
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("create instance: %w", err)
	}
	meta.SetExternalName(cr, i.ID)
	return managed.ExternalCreation{}, nil
}

func (e *instanceExternal) Update(ctx context.Context, cr *v1alpha1.Instance) (managed.ExternalUpdate, error) {
	// Resizing a provisioning instance is refused by the cloud; returning the
	// error makes the reconciler retry once it is running.
	if _, err := e.cloud.ResizeInstance(ctx, meta.GetExternalName(cr), cr.Spec.ForProvider.Size); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("resize instance: %w", err)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *instanceExternal) Delete(ctx context.Context, cr *v1alpha1.Instance) (managed.ExternalDelete, error) {
	cr.SetConditions(xpv2.Deleting())
	if err := e.cloud.DeleteInstance(ctx, meta.GetExternalName(cr)); err != nil && !clients.IsNotFound(err) {
		return managed.ExternalDelete{}, fmt.Errorf("delete instance: %w", err)
	}
	return managed.ExternalDelete{}, nil
}

func (e *instanceExternal) Disconnect(context.Context) error { return nil }
