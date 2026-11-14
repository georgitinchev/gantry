package v1alpha1

import xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"

// Methods that make Network and Instance satisfy resource.ModernManaged.
// Written by hand rather than generated; they only forward to embedded fields.

func (n *Network) GetCondition(ct xpv2.ConditionType) xpv2.Condition {
	return n.Status.GetCondition(ct)
}
func (n *Network) SetConditions(c ...xpv2.Condition)               { n.Status.SetConditions(c...) }
func (n *Network) GetManagementPolicies() xpv2.ManagementPolicies  { return n.Spec.ManagementPolicies }
func (n *Network) SetManagementPolicies(p xpv2.ManagementPolicies) { n.Spec.ManagementPolicies = p }
func (n *Network) GetProviderConfigReference() *xpv2.ProviderConfigReference {
	return n.Spec.ProviderConfigReference
}
func (n *Network) SetProviderConfigReference(r *xpv2.ProviderConfigReference) {
	n.Spec.ProviderConfigReference = r
}
func (n *Network) GetWriteConnectionSecretToReference() *xpv2.LocalSecretReference {
	return n.Spec.WriteConnectionSecretToReference
}
func (n *Network) SetWriteConnectionSecretToReference(r *xpv2.LocalSecretReference) {
	n.Spec.WriteConnectionSecretToReference = r
}

func (i *Instance) GetCondition(ct xpv2.ConditionType) xpv2.Condition {
	return i.Status.GetCondition(ct)
}
func (i *Instance) SetConditions(c ...xpv2.Condition)               { i.Status.SetConditions(c...) }
func (i *Instance) GetManagementPolicies() xpv2.ManagementPolicies  { return i.Spec.ManagementPolicies }
func (i *Instance) SetManagementPolicies(p xpv2.ManagementPolicies) { i.Spec.ManagementPolicies = p }
func (i *Instance) GetProviderConfigReference() *xpv2.ProviderConfigReference {
	return i.Spec.ProviderConfigReference
}
func (i *Instance) SetProviderConfigReference(r *xpv2.ProviderConfigReference) {
	i.Spec.ProviderConfigReference = r
}
func (i *Instance) GetWriteConnectionSecretToReference() *xpv2.LocalSecretReference {
	return i.Spec.WriteConnectionSecretToReference
}
func (i *Instance) SetWriteConnectionSecretToReference(r *xpv2.LocalSecretReference) {
	i.Spec.WriteConnectionSecretToReference = r
}
