<div align='center'>
<img src="assets/logo.png" width=50%  height=200px>
<hr>

### **certificate-management-operator**
A controller for Certificate resources that integrates with cert-manager or handles internal PKI, automatically rotating certificates, updating secrets, and triggering rolling restarts of affected deployments. Relevant for security-focused roles.

![Code Size](https://img.shields.io/github/languages/code-size/namansharma18899/certificate-management-operator)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/namansharma18899/certificate-management-operator/blob/main/LICENSE)
[![GitHub forks](https://img.shields.io/github/forks/namansharma18899/certificate-management-operator)](https://github.com/namansharma18899/certificate-management-operator/network)
[![GitHub issues](https://img.shields.io/github/issues/namansharma18899/certificate-management-operator)](https://github.com/namansharma18899/certificate-management-operator/issues)
[![Last Commit](https://img.shields.io/github/last-commit/namansharma18899/certificate-management-operator)](https://github.com/namansharma18899/certificate-management-operator/commits/main)

*If you like my work a ⭐ would mean the world*
</div>

---

## 🤨 Why
- No one likes to manage certificates !! They are a pain & they not a sec should be spent fixing cron jobs to update them !!!

## 🧪 Create a test cluster and run locally

### Option A: Kind (recommended)

**Prerequisites:** Docker must be installed and **running** (e.g. start Docker Desktop). Kind runs cluster nodes as Docker containers.

1. **Install Kind** (if not already): https://kind.sigs.k8s.io/docs/user/quick-start/#installation

2. **Create a cluster:**
   ```bash
   kind create cluster --name cert-operator-dev
   ```
   Your `kubectl` will now point at this cluster (check with `kubectl cluster-info`).

3. **Install the CRDs and run the controller locally:**
   ```bash
   cd /path/to/certificate-management-operator
   make install          # installs Certificate CRD into the cluster
   make run              # runs the controller on your machine, talking to the cluster
   ```

4. **In another terminal**, create a Certificate and verify:
   ```bash
   kubectl apply -f config/samples/cert_v1alpha1_certificate.yaml
   kubectl get certificates
   kubectl describe certificate certificate-sample
   kubectl get secret myapp-tls
   kubectl get secret myapp-tls -o yaml
   ```

5. **Tear down the cluster when done:**
   ```bash
   kind delete cluster --name cert-operator-dev
   ```

### Option B: Minikube

1. **Start a cluster:** `minikube start`
2. **Install and run** (same as above): `make install` then `make run` in the project directory.
3. **Apply sample and verify** as in step 4 above.
4. **Stop:** `minikube stop` or `minikube delete`.

### Run controller locally (any cluster)

- Ensure your kubeconfig points at your test cluster: `kubectl config current-context`.
- From the project root: `make install` (once) then `make run`.
- The controller runs on your machine and reconciles `Certificate` resources in the cluster; no need to build or push an image.

## ᛜ Build and Deploy on Openshift
```bash
$ export IMG=quay.io/yourusername/certificate-operator:v0.1.0
$ make docker-build docker-push IMG=$IMG
$ make deploy IMG=$IMG
$ oc get deployment -n certificate-management-operator-system
$ oc get pods -n certificate-management-operator-system
$ oc logs -n certificate-management-operator-system deployment/certificate-management-operator-controller-manager
```

## 🤗 Support
- Make sure to leave a ⭐ if you like this project.
