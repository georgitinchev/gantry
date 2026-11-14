//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.23.0 object paths=./apis/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.23.0 crd paths=./apis/... output:crd:artifacts:config=package/crds

package provider
