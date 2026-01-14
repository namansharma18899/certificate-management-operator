package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IssuerRef references a certificate issuer
type IssuerRef struct {
	// Name of the issuer
	Name string `json:"name"`

	// Kind of the issuer (SelfSigned, CA, External)
	// +optional
	// +kubebuilder:default=SelfSigned
	Kind string `json:"kind,omitempty"`
}

// CertificateSpec defines the desired state of Certificate
type CertificateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// CommonName is the CN for the certificate
	// +kubebuilder:validation:Required
	CommonName string `json:"commonName"`

	// DNSNames is a list of DNS subject alternative names
	// +optional
	DNSNames []string `json:"dnsNames,omitempty"`

	// IPAddresses is a list of IP subject alternative names
	// +optional
	IPAddresses []string `json:"ipAddresses,omitempty"`

	// SecretName where the certificate will be stored
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`

	// Duration for certificate validity (e.g., "2160h" for 90 days)
	// +optional
	// +kubebuilder:default="2160h"
	Duration string `json:"duration,omitempty"`

	// RenewBefore specifies when to renew (e.g., "720h" for 30 days before expiry)
	// +optional
	// +kubebuilder:default="720h"
	RenewBefore string `json:"renewBefore,omitempty"`

	// IssuerRef references the certificate issuer
	// +optional
	IssuerRef IssuerRef `json:"issuerRef,omitempty"`

	// RestartDeployments triggers restart of deployments using this cert
	// +optional
	RestartDeployments bool `json:"restartDeployments,omitempty"`
}

// CertificateStatus defines the observed state of Certificate
type CertificateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions represent the latest available observations of an object's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// NotBefore is the certificate start time
	// +optional
	NotBefore *metav1.Time `json:"notBefore,omitempty"`

	// NotAfter is the certificate expiry time
	// +optional
	NotAfter *metav1.Time `json:"notAfter,omitempty"`

	// RenewalTime is when the certificate should be renewed
	// +optional
	RenewalTime *metav1.Time `json:"renewalTime,omitempty"`

	// SerialNumber of the current certificate
	// +optional
	SerialNumber string `json:"serialNumber,omitempty"`

	// LastRenewalTime is when the certificate was last renewed
	// +optional
	LastRenewalTime *metav1.Time `json:"lastRenewalTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=cert;certs
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="Certificate ready status"
//+kubebuilder:printcolumn:name="Secret",type="string",JSONPath=".spec.secretName",description="Secret name"
//+kubebuilder:printcolumn:name="Issuer",type="string",JSONPath=".spec.issuerRef.name",description="Issuer name"
//+kubebuilder:printcolumn:name="Expiry",type="date",JSONPath=".status.notAfter",description="Certificate expiry time"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Certificate is the Schema for the certificates API
type Certificate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertificateSpec   `json:"spec,omitempty"`
	Status CertificateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CertificateList contains a list of Certificate
type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Certificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Certificate{}, &CertificateList{})
}
