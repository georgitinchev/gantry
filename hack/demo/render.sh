#!/usr/bin/env bash
# Renders the visitor's choices into request files. Inputs were already
# restricted to fixed choices by the workflow; they are re-checked here anyway.
# env: DEMO_ID SIZE EXPOSURE POSTGRES VERSION
set -euo pipefail
cd "$(dirname "$0")/../.."
[[ "$DEMO_ID" =~ ^[a-f0-9]{16}$ ]] || { echo "bad demo id"; exit 1; }
case "$SIZE" in small|medium) ;; *) echo "bad size"; exit 1;; esac
case "$EXPOSURE" in internal|public) ;; *) echo "bad exposure"; exit 1;; esac
case "$POSTGRES" in true|false) ;; *) echo "bad postgres flag"; exit 1;; esac
case "$VERSION" in 16|17|18) ;; *) echo "bad version"; exit 1;; esac

env="demo-${DEMO_ID:0:8}"
mkdir -p requests/demo
cat > requests/demo/appenvironment.yaml <<YAML
apiVersion: platform.gantry.internal/v1alpha1
kind: AppEnvironment
metadata:
  name: ${env}
spec:
  team: demo
  size: ${SIZE}
  exposure: ${EXPOSURE}
YAML
if [[ "$POSTGRES" == true ]]; then
cat > requests/demo/postgres-db.yaml <<YAML
apiVersion: platform.gantry.internal/v1alpha1
kind: PostgresInstance
metadata:
  name: db
  namespace: ${env}
spec:
  size: small
  version: "${VERSION}"
YAML
fi
echo "$env"
