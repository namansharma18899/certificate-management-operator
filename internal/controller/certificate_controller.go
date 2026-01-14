package controller

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	certv1alpha1 "github.com/namansharma18899/certificate-management-operator/api/v1alpha1"
)

const (
	certificateFinalizer = "cert.example.com/finalizer"
	typeAvailableCert    = "Available"
	typeReadyCert        = "Ready"
)

// CertificateReconciler reconciles a Certificate object
type CertificateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cert.example.com,resources=certificates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cert.example.com,resources=certificates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cert.example.com,resources=certificates/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Certificate")

	// Fetch the Certificate instance
	certificate := &certv1alpha1.Certificate{}
	err := r.Get(ctx, req.NamespacedName, certificate)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Certificate resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Certificate")
		return ctrl.Result{}, err
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(certificate, certificateFinalizer) {
		logger.Info("Adding Finalizer for Certificate")
		if ok := controllerutil.AddFinalizer(certificate, certificateFinalizer); !ok {
			logger.Error(err, "Failed to add finalizer to Certificate")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, certificate); err != nil {
			logger.Error(err, "Failed to update Certificate with finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the Certificate instance is marked to be deleted
	if certificate.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(certificate, certificateFinalizer) {
			logger.Info("Performing cleanup for Certificate")

			// Remove finalizer
			if ok := controllerutil.RemoveFinalizer(certificate, certificateFinalizer); !ok {
				logger.Error(err, "Failed to remove finalizer from Certificate")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, certificate); err != nil {
				logger.Error(err, "Failed to remove finalizer from Certificate")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Check if certificate needs renewal
	if r.needsRenewal(certificate) {
		logger.Info("Certificate needs issuance or renewal", "name", certificate.Name)

		// Generate new certificate
		certPEM, keyPEM, notBefore, notAfter, serialNumber, err := r.generateCertificate(certificate)
		if err != nil {
			logger.Error(err, "Failed to generate certificate")
			meta.SetStatusCondition(&certificate.Status.Conditions, metav1.Condition{
				Type:               typeReadyCert,
				Status:             metav1.ConditionFalse,
				Reason:             "GenerationFailed",
				Message:            fmt.Sprintf("Failed to generate certificate: %v", err),
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, certificate); err != nil {
				logger.Error(err, "Failed to update Certificate status")
			}
			return ctrl.Result{}, err
		}

		// Create or update secret
		err = r.createOrUpdateSecret(ctx, certificate, certPEM, keyPEM)
		if err != nil {
			logger.Error(err, "Failed to create/update secret")
			meta.SetStatusCondition(&certificate.Status.Conditions, metav1.Condition{
				Type:               typeReadyCert,
				Status:             metav1.ConditionFalse,
				Reason:             "SecretUpdateFailed",
				Message:            fmt.Sprintf("Failed to update secret: %v", err),
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, certificate); err != nil {
				logger.Error(err, "Failed to update Certificate status")
			}
			return ctrl.Result{}, err
		}

		// Update status
		certificate.Status.NotBefore = &metav1.Time{Time: notBefore}
		certificate.Status.NotAfter = &metav1.Time{Time: notAfter}
		certificate.Status.RenewalTime = r.calculateRenewalTime(certificate, notAfter)
		certificate.Status.SerialNumber = serialNumber
		certificate.Status.LastRenewalTime = &metav1.Time{Time: time.Now()}

		// Set Ready condition
		meta.SetStatusCondition(&certificate.Status.Conditions, metav1.Condition{
			Type:               typeReadyCert,
			Status:             metav1.ConditionTrue,
			Reason:             "CertificateIssued",
			Message:            "Certificate has been issued successfully",
			LastTransitionTime: metav1.Now(),
		})

		meta.SetStatusCondition(&certificate.Status.Conditions, metav1.Condition{
			Type:               typeAvailableCert,
			Status:             metav1.ConditionTrue,
			Reason:             "Reconciling",
			Message:            fmt.Sprintf("Certificate for (%s) issued successfully", certificate.Name),
			LastTransitionTime: metav1.Now(),
		})

		if err := r.Status().Update(ctx, certificate); err != nil {
			logger.Error(err, "Failed to update Certificate status")
			return ctrl.Result{}, err
		}

		// Restart deployments if enabled
		if certificate.Spec.RestartDeployments {
			if err := r.restartDeployments(ctx, certificate); err != nil {
				logger.Error(err, "Failed to restart deployments")
				// Don't fail the reconciliation, just log the error
			}
		}

		logger.Info("Certificate issued successfully", "name", certificate.Name, "notAfter", notAfter)
	}

	// Requeue before renewal time
	requeueAfter := r.getRequeueTime(certificate)
	logger.Info("Requeuing reconciliation", "after", requeueAfter)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// needsRenewal checks if certificate needs to be issued or renewed
func (r *CertificateReconciler) needsRenewal(cert *certv1alpha1.Certificate) bool {
	// If no renewal time set, needs initial issuance
	if cert.Status.RenewalTime == nil {
		return true
	}

	// Check if current time is past renewal time
	return time.Now().After(cert.Status.RenewalTime.Time)
}

// generateCertificate creates a new self-signed certificate
func (r *CertificateReconciler) generateCertificate(cert *certv1alpha1.Certificate) ([]byte, []byte, time.Time, time.Time, string, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, time.Time{}, time.Time{}, "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Parse duration (default to 90 days)
	duration := 90 * 24 * time.Hour
	if cert.Spec.Duration != "" {
		duration, err = time.ParseDuration(cert.Spec.Duration)
		if err != nil {
			return nil, nil, time.Time{}, time.Time{}, "", fmt.Errorf("invalid duration: %w", err)
		}
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(duration)

	// Generate serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, time.Time{}, time.Time{}, "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Parse IP addresses
	var ipAddresses []net.IP
	for _, ipStr := range cert.Spec.IPAddresses {
		if ip := net.ParseIP(ipStr); ip != nil {
			ipAddresses = append(ipAddresses, ip)
		}
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cert.Spec.CommonName,
			Organization: []string{"Certificate Operator"},
		},
		DNSNames:              cert.Spec.DNSNames,
		IPAddresses:           ipAddresses,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, time.Time{}, time.Time{}, "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key to PEM
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return certPEM, keyPEM, notBefore, notAfter, fmt.Sprintf("%x", serialNumber), nil
}

// createOrUpdateSecret creates or updates the TLS secret
func (r *CertificateReconciler) createOrUpdateSecret(ctx context.Context, cert *certv1alpha1.Certificate, certPEM, keyPEM []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cert.Spec.SecretName,
			Namespace: cert.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "certificate-operator",
				"cert.example.com/certificate": cert.Name,
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": keyPEM,
		},
	}

	// Set owner reference
	if err := ctrl.SetControllerReference(cert, secret, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Try to get existing secret
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, existingSecret)

	if err != nil && errors.IsNotFound(err) {
		// Create new secret
		return r.Create(ctx, secret)
	} else if err != nil {
		return err
	}

	// Update existing secret
	existingSecret.Data = secret.Data
	existingSecret.Labels = secret.Labels
	return r.Update(ctx, existingSecret)
}

// calculateRenewalTime determines when the certificate should be renewed
func (r *CertificateReconciler) calculateRenewalTime(cert *certv1alpha1.Certificate, notAfter time.Time) *metav1.Time {
	// Default to 30 days before expiry
	renewBefore := 30 * 24 * time.Hour

	if cert.Spec.RenewBefore != "" {
		duration, err := time.ParseDuration(cert.Spec.RenewBefore)
		if err == nil {
			renewBefore = duration
		}
	}

	renewalTime := notAfter.Add(-renewBefore)
	return &metav1.Time{Time: renewalTime}
}

// getRequeueTime calculates when to requeue the reconciliation
func (r *CertificateReconciler) getRequeueTime(cert *certv1alpha1.Certificate) time.Duration {
	if cert.Status.RenewalTime == nil {
		return time.Minute
	}

	timeUntilRenewal := time.Until(cert.Status.RenewalTime.Time)
	if timeUntilRenewal < 0 {
		return time.Minute
	}

	// Requeue 1 hour before renewal time, or if less than 1 hour remains, requeue in half that time
	if timeUntilRenewal < time.Hour {
		return timeUntilRenewal / 2
	}

	return timeUntilRenewal - time.Hour
}

// restartDeployments triggers rolling restart of deployments using this certificate
func (r *CertificateReconciler) restartDeployments(ctx context.Context, cert *certv1alpha1.Certificate) error {
	logger := log.FromContext(ctx)

	// List all deployments in the namespace
	deployments := &appsv1.DeploymentList{}
	if err := r.List(ctx, deployments, client.InNamespace(cert.Namespace)); err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	restartedCount := 0
	for i := range deployments.Items {
		deploy := &deployments.Items[i]

		// Check if deployment uses this secret
		if r.deploymentUsesSecret(deploy, cert.Spec.SecretName) {
			logger.Info("Restarting deployment", "deployment", deploy.Name)

			// Trigger rolling restart by updating annotation
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = make(map[string]string)
			}
			deploy.Spec.Template.Annotations["cert.example.com/restartedAt"] = time.Now().Format(time.RFC3339)

			if err := r.Update(ctx, deploy); err != nil {
				logger.Error(err, "Failed to restart deployment", "deployment", deploy.Name)
				continue
			}
			restartedCount++
		}
	}

	logger.Info("Deployment restart completed", "count", restartedCount)
	return nil
}

// deploymentUsesSecret checks if a deployment references a specific secret
func (r *CertificateReconciler) deploymentUsesSecret(deploy *appsv1.Deployment, secretName string) bool {
	// Check volumes
	for _, volume := range deploy.Spec.Template.Spec.Volumes {
		if volume.Secret != nil && volume.Secret.SecretName == secretName {
			return true
		}
	}

	// Check environment variables from secrets
	for _, container := range deploy.Spec.Template.Spec.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil && envFrom.SecretRef.Name == secretName {
				return true
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				if env.ValueFrom.SecretKeyRef.Name == secretName {
					return true
				}
			}
		}
	}

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certv1alpha1.Certificate{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
