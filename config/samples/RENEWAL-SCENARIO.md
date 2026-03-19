# Renewal + Deployment Restart Scenario

This scenario shows the controller **renewing** a short-lived certificate and **restarting** the deployment that uses it (when `restartDeployments: true`).

## What happens

1. You apply a Certificate with **short validity** (`duration: 5m`, `renewBefore: 2m`).
2. The controller issues the cert and creates the TLS secret; the deployment starts and serves HTTPS.
3. About **3 minutes** later, the controller sees that renewal time has passed, issues a new cert, updates the secret, and triggers a **rolling restart** of the deployment (via the `cert.example.com/restartedAt` annotation).
4. Pods pick up the new certificate from the updated secret.

## Prerequisites

- Cluster with the certificate operator running (CRD installed, controller in the same namespace as the resources).
- `kubectl` pointed at that cluster.

## Steps

### 1. Create a namespace and apply the scenario

```bash
# Use default namespace or create one, e.g.:
kubectl create namespace cert-demo --dry-run=client -o yaml | kubectl apply -f -
kubectl config set-context --current --namespace=cert-demo

# Install CRD if not already (from repo root)
kubectl apply -f config/crd/bases/cert.example.com_certificates.yaml

# Apply the renewal scenario (Certificate + Deployment + Service)
kubectl apply -f config/samples/renewal-scenario.yaml
```

### 2. Confirm cert issued and deployment running

```bash
# Certificate should show Ready=True and a Secret name
kubectl get certificates
kubectl get secret renewal-demo-tls

# Deployment and pod should be Ready
kubectl get deployment renewal-demo
kubectl get pods -l app=renewal-demo
```

### 3. Watch controller logs (optional)

In another terminal, follow the operator logs so you can see renewal and restart:

```bash
# If the operator runs as a deployment in the same namespace:
kubectl logs -f deployment/certificate-management-operator-controller-manager -c manager
```

### 4. Wait for renewal (~3 minutes)

- Certificate is valid for **5 minutes**, with **renew 2 minutes before expiry**.
- So renewal time is about **3 minutes** after first issuance.
- The controller requeues before that; when the clock passes renewal time, the next reconcile will renew and restart.

### 5. Verify renewal and restart

```bash
# Certificate status: NotAfter and LastRenewalTime should update
kubectl get certificate certificate-renewal-demo -o yaml

# Secret should have new content (timestamp will change)
kubectl get secret renewal-demo-tls -o jsonpath='{.metadata.resourceVersion}'

# Deployment should have cert.example.com/restartedAt on the pod template
kubectl get deployment renewal-demo -o jsonpath='{.spec.template.metadata.annotations}'

# New pod (rolling restart)
kubectl get pods -l app=renewal-demo -w
```

### 6. Optional: hit the service

```bash
kubectl port-forward svc/renewal-demo 8444:8443
curl -k https://localhost:8444/
# Expected: OK - renewed cert
```

## Cleanup

```bash
kubectl delete -f config/samples/renewal-scenario.yaml
```
