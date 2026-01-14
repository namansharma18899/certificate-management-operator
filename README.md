# certificate-management-operator
A controller for Certificate resources that integrates with cert-manager or handles internal PKI, automatically rotating certificates, updating secrets, and triggering rolling restarts of affected deployments. Relevant for security-focused roles.

<hr>

![Code Size](https://img.shields.io/github/languages/code-size/namansharma18899/certificate-management-operator)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/namansharma18899/certificate-management-operator/blob/main/LICENSE)
[![GitHub forks](https://img.shields.io/github/forks/namansharma18899/certificate-management-operator)](https://github.com/namansharma18899/certificate-management-operator/network)
[![GitHub issues](https://img.shields.io/github/issues/namansharma18899/certificate-management-operator)](https://github.com/namansharma18899/certificate-management-operator/issues)
[![Last Commit](https://img.shields.io/github/last-commit/namansharma18899/certificate-management-operator)](https://github.com/namansharma18899/certificate-management-operator/commits/main)

*If you like my work a ⭐ would mean the world*

---

## 🤨 Why
- No one likes to manage certificates !! They are a pain & they not a sec should be spent fixing cron jobs to update them !!!

## ⚙️ Test Locally with Kind or Minikube
```bash
$ make install
$ make run
$ kubectl apply -f config/samples/cert_v1alpha1_certificate.yaml
$ kubectl get certificates
$ kubectl describe certificate certificate-sample
$ kubectl get secret myapp-tls
$ kubectl get secret myapp-tls -o yaml
```

## ᛜ Build and Deploy on Openshift
```bash
$ export IMG=quay.io/yourusername/certificate-operator:v0.1.0
$ make docker-build docker-push IMG=$IMG
$ make deploy IMG=$IMG
$ oc get deployment -n certificate-operator-system
$ oc get pods -n certificate-operator-system
$ oc logs -n certificate-operator-system deployment/certificate-operator-controller-manager
```

## 🤗 Support
- Make sure to leave a ⭐ if you like this project.
