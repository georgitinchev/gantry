package controller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/georgitinchev/gantry/provider/apis/v1alpha1"
	"github.com/georgitinchev/gantry/provider/internal/clients"
)

// Cloud is the subset of the gantry-cloud client the controllers use; tests fake it.
type Cloud interface {
	GetNetwork(ctx context.Context, id string) (*clients.Network, error)
	CreateNetwork(ctx context.Context, name, cidr string) (*clients.Network, error)
	DeleteNetwork(ctx context.Context, id string) error
	GetInstance(ctx context.Context, id string) (*clients.Instance, error)
	CreateInstance(ctx context.Context, name, networkID, size string) (*clients.Instance, error)
	ResizeInstance(ctx context.Context, id, size string) (*clients.Instance, error)
	DeleteInstance(ctx context.Context, id string) error
}

// cloudName makes the external name unique across namespaces and stable
// across retries, which is what makes Create idempotent.
func cloudName(namespace, name string) string { return namespace + "." + name }

type networkExternal struct{ cloud Cloud }

func (e *networkExternal) Observe(ctx context.Context, cr *v1alpha1.Network) (managed.ExternalObservation, error) {
	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	n, err := e.cloud.GetNetwork(ctx, id)
	if clients.IsNotFound(err) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("observe network: %w", err)
	}
	cr.Status.AtProvider = v1alpha1.NetworkObservation{ID: n.ID, CIDR: n.CIDR}
	cr.SetConditions(xpv2.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true, // cidr is immutable (CEL), nothing to update
		ConnectionDetails: managed.ConnectionDetails{
			"cidr": []byte(n.CIDR),
		},
	}, nil
}

func (e *networkExternal) Create(ctx context.Context, cr *v1alpha1.Network) (managed.ExternalCreation, error) {
	cr.SetConditions(xpv2.Creating())
	n, err := e.cloud.CreateNetwork(ctx, cloudName(cr.Namespace, cr.Name), cr.Spec.ForProvider.CIDR)
	if apiErr, conflict := clients.IsConflict(err); conflict {
		// We created it before but lost the external-name update: adopt it.
		existing := &clients.Network{}
		if jerr := json.Unmarshal(apiErr.Existing, existing); jerr != nil || existing.ID == "" {
			return managed.ExternalCreation{}, fmt.Errorf("create network: conflict without existing object: %w", err)
		}
		n, err = existing, nil
	}
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("create network: %w", err)
	}
	meta.SetExternalName(cr, n.ID)
	return managed.ExternalCreation{}, nil
}

func (e *networkExternal) Update(context.Context, *v1alpha1.Network) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *networkExternal) Delete(ctx context.Context, cr *v1alpha1.Network) (managed.ExternalDelete, error) {
	cr.SetConditions(xpv2.Deleting())
	err := e.cloud.DeleteNetwork(ctx, meta.GetExternalName(cr))
	if err != nil && !clients.IsNotFound(err) {
		// in_use while instances still exist: the reconciler retries with backoff.
		return managed.ExternalDelete{}, fmt.Errorf("delete network: %w", err)
	}
	return managed.ExternalDelete{}, nil
}

func (e *networkExternal) Disconnect(context.Context) error { return nil }
