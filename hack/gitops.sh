#!/usr/bin/env bash
# Installs Argo CD and an in-cluster Git server, pushes the commit under test,
# and hands platform APIs, policies and every team's requests to Argo CD.
set -euo pipefail
cd "$(dirname "$0")/.."
source hack/versions.env
log() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }

log "in-cluster git server"
kubectl apply -f gitops/git-server/git-server.yaml
kubectl -n gitops rollout status deploy/git-server --timeout=120s

log "push the commit under test"
# Snapshot the whole working tree (new files included, .gitignore respected) through
# a temporary index, so local runs test exactly what is on disk and the real index is untouched.
tmp_index=$(mktemp)
GIT_INDEX_FILE=$tmp_index git read-tree HEAD
GIT_INDEX_FILE=$tmp_index git add -A
tree=$(GIT_INDEX_FILE=$tmp_index git write-tree)
rm -f "$tmp_index"
rev=$(GIT_AUTHOR_NAME=gantry GIT_AUTHOR_EMAIL=ci@gantry.internal GIT_COMMITTER_NAME=gantry GIT_COMMITTER_EMAIL=ci@gantry.internal \
  git commit-tree "$tree" -m "tree under test: $(git rev-parse --short HEAD)")
# A parentless commit: Argo CD needs the tree, not the history, and it can be
# pushed from CI's shallow checkout (pushing HEAD's ancestry would be refused).
kubectl -n gitops port-forward svc/git-server 19418:9418 >/dev/null 2>&1 &
pf=$!; trap 'kill $pf 2>/dev/null' EXIT
for _ in $(seq 1 20); do nc -z 127.0.0.1 19418 2>/dev/null && break; sleep 0.5; done
git push -q --force git://127.0.0.1:19418/gantry.git "$rev:refs/heads/main"
echo "pushed $(git rev-parse --short "$rev")"

log "argo cd ${ARGOCD_CHART}"
helm repo add --force-update argo https://argoproj.github.io/argo-helm >/dev/null
helm repo update argo >/dev/null
helm upgrade --install argocd argo/argo-cd --version "${ARGOCD_CHART}" \
  -n argocd --create-namespace -f gitops/argocd-values.yaml --wait --timeout 6m

log "app of apps"
kubectl apply -f gitops/root.yaml
for app in platform-apis policies; do
  until kubectl -n argocd get application "$app" >/dev/null 2>&1; do sleep 3; done
done
kubectl -n argocd wait application platform-apis policies --for=jsonpath='{.status.health.status}'=Healthy --timeout=300s
kubectl wait xrd --all --for=condition=Established --timeout=120s
kubectl wait validatingpolicies.policies.kyverno.io --all --for=jsonpath='{.status.conditionStatus.ready}'=true --timeout=120s
log "gitops ready"
