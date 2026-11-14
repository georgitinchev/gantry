package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/georgitinchev/gantry/provider/apis/v1alpha1"
	"github.com/georgitinchev/gantry/provider/internal/clients"
)

type fakeCloud struct {
	networks  map[string]*clients.Network
	instances map[string]*clients.Instance
	created   int
}

func newFake() *fakeCloud {
	return &fakeCloud{networks: map[string]*clients.Network{}, instances: map[string]*clients.Instance{}}
}

func notFound() error { return &clients.APIError{Status: http.StatusNotFound, Code: "not_found"} }
func conflict(v any) error {
	b, _ := json.Marshal(v)
	return &clients.APIError{Status: http.StatusConflict, Code: "conflict", Existing: b}
}

func (f *fakeCloud) GetNetwork(_ context.Context, id string) (*clients.Network, error) {
	if n, ok := f.networks[id]; ok {
		return n, nil
	}
	return nil, notFound()
}
func (f *fakeCloud) CreateNetwork(_ context.Context, name, cidr string) (*clients.Network, error) {
	for _, n := range f.networks {
		if n.Name == name {
			return nil, conflict(n)
		}
	}
	f.created++
	n := &clients.Network{ID: "net-1", Name: name, CIDR: "10.40.0.0/24"}
	f.networks[n.ID] = n
	return n, nil
}
func (f *fakeCloud) DeleteNetwork(_ context.Context, id string) error {
	if _, ok := f.networks[id]; !ok {
		return notFound()
	}
	delete(f.networks, id)
	return nil
}
func (f *fakeCloud) GetInstance(_ context.Context, id string) (*clients.Instance, error) {
	if i, ok := f.instances[id]; ok {
		return i, nil
	}
	return nil, notFound()
}
func (f *fakeCloud) CreateInstance(_ context.Context, name, netID, size string) (*clients.Instance, error) {
	f.created++
	i := &clients.Instance{ID: "vm-1", Name: name, NetworkID: netID, Size: size, IP: "10.40.0.10", State: "provisioning"}
	f.instances[i.ID] = i
	return i, nil
}
func (f *fakeCloud) ResizeInstance(_ context.Context, id, size string) (*clients.Instance, error) {
	i := f.instances[id]
	if i.State != "running" {
		return nil, &clients.APIError{Status: http.StatusConflict, Code: "not_running"}
	}
	i.Size, i.State = size, "resizing"
	return i, nil
}
func (f *fakeCloud) DeleteInstance(_ context.Context, id string) error {
	if _, ok := f.instances[id]; !ok {
		return notFound()
	}
	delete(f.instances, id)
	return nil
}

func network(name string) *v1alpha1.Network {
	return &v1alpha1.Network{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team-dev"}}
}

func TestNetworkLifecycle(t *testing.T) {
	ctx := context.Background()
	cloud := newFake()
	e := &networkExternal{cloud: cloud}
	cr := network("net")

	obs, err := e.Observe(ctx, cr)
	if err != nil || obs.ResourceExists {
		t.Fatalf("no external name: want not exists, got %+v %v", obs, err)
	}
	if _, err := e.Create(ctx, cr); err != nil {
		t.Fatal(err)
	}
	if meta.GetExternalName(cr) != "net-1" {
		t.Fatalf("external name not set: %q", meta.GetExternalName(cr))
	}
	obs, _ = e.Observe(ctx, cr)
	if !obs.ResourceExists || !obs.ResourceUpToDate || cr.Status.AtProvider.CIDR != "10.40.0.0/24" {
		t.Fatalf("observe after create: %+v %+v", obs, cr.Status.AtProvider)
	}
	if cr.GetCondition(xpv2.TypeReady).Status != corev1.ConditionTrue {
		t.Fatal("want Ready=True")
	}
	if _, err := e.Delete(ctx, cr); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Delete(ctx, cr); err != nil {
		t.Fatalf("delete must be idempotent, got %v", err)
	}
}

func TestNetworkCreateAdoptsAfterLostExternalName(t *testing.T) {
	ctx := context.Background()
	cloud := newFake()
	e := &networkExternal{cloud: cloud}
	first := network("net")
	e.Create(ctx, first)

	// Same object, but the external-name annotation write was lost.
	retry := network("net")
	if _, err := e.Create(ctx, retry); err != nil {
		t.Fatalf("retry should adopt, got %v", err)
	}
	if meta.GetExternalName(retry) != "net-1" || cloud.created != 1 {
		t.Fatalf("want adoption of net-1 without a second create, got %q (created %d)", meta.GetExternalName(retry), cloud.created)
	}
}

func TestInstanceWaitsForNetworkThenProvisions(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	net := network("net")
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(net).WithStatusSubresource(&v1alpha1.Network{}).Build()
	cloud := newFake()
	e := &instanceExternal{cloud: cloud, kube: kube}
	cr := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "vm", Namespace: "team-dev"},
		Spec:       v1alpha1.InstanceSpec{ForProvider: v1alpha1.InstanceParameters{Size: "small", NetworkRef: v1alpha1.LocalRef{Name: "net"}}},
	}

	if _, err := e.Create(ctx, cr); !errors.Is(err, errNetworkNotReady) {
		t.Fatalf("want errNetworkNotReady before the network has an ID, got %v", err)
	}

	net.Status.AtProvider.ID = "net-1"
	if err := kube.Status().Update(ctx, net); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Create(ctx, cr); err != nil {
		t.Fatal(err)
	}
	obs, _ := e.Observe(ctx, cr)
	if cr.GetCondition(xpv2.TypeReady).Reason != xpv2.ReasonCreating || cr.Status.AtProvider.IP != "10.40.0.10" || !obs.ResourceUpToDate {
		t.Fatalf("provisioning instance: %+v %+v", cr.Status.AtProvider, cr.GetCondition(xpv2.TypeReady))
	}

	cloud.instances["vm-1"].State = "running"
	e.Observe(ctx, cr)
	if cr.GetCondition(xpv2.TypeReady).Status != corev1.ConditionTrue {
		t.Fatal("running instance should be Ready")
	}
}

func TestInstanceResize(t *testing.T) {
	ctx := context.Background()
	cloud := newFake()
	cloud.instances["vm-1"] = &clients.Instance{ID: "vm-1", Size: "small", State: "provisioning"}
	e := &instanceExternal{cloud: cloud}
	cr := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "vm", Namespace: "team-dev", Annotations: map[string]string{meta.AnnotationKeyExternalName: "vm-1"}},
		Spec:       v1alpha1.InstanceSpec{ForProvider: v1alpha1.InstanceParameters{Size: "medium"}},
	}
	if obs, _ := e.Observe(ctx, cr); obs.ResourceUpToDate {
		t.Fatal("size drift must report not up to date")
	}
	if _, err := e.Update(ctx, cr); err == nil {
		t.Fatal("resize while provisioning should error so the reconciler retries")
	}
	cloud.instances["vm-1"].State = "running"
	if _, err := e.Update(ctx, cr); err != nil {
		t.Fatal(err)
	}
	if cloud.instances["vm-1"].Size != "medium" {
		t.Fatal("resize not applied")
	}
}
