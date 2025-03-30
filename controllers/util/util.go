package util

import (
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/j-be/desec-dns-operator/api/v1"
)

func GetSubnames(ingress networkingv1.Ingress, domain string) []string {
	suffix := "." + domain

	subnames := []string{}
	for _, rule := range ingress.Spec.Rules {
		host := strings.TrimRight(rule.Host, ".")
		if strings.HasSuffix(host, suffix) {
			subnames = append(subnames, strings.TrimSuffix(host, suffix))
		}
	}
	return subnames
}

func GetIps(ingress networkingv1.Ingress) []string {
	ips := []string{}
	for _, ingress := range ingress.Status.LoadBalancer.Ingress {
		ips = append(ips, ingress.IP)
	}
	return ips
}

func InitializeDesecDns(namespacedName types.NamespacedName) *v1.DesecDns {
	cr := new(v1.DesecDns)
	cr.Name = namespacedName.Name
	cr.Namespace = namespacedName.Namespace
	cr.Spec.IPs = []string{}
	return cr
}

func InitializeDesecDnsStatus() v1.DesecDnsStatus {
	status := v1.DesecDnsStatus{
		Conditions: []metav1.Condition{},
	}
	for _, conditionType := range []string{"Domain", "IpUpdate"} {
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:   conditionType,
			Status: metav1.ConditionUnknown,
			Reason: "Initializing",
		})
	}

	return status
}

func UpdateDesecDnsStatus(
	status *v1.DesecDnsStatus,
	conditionType string,
	conditionStatus metav1.ConditionStatus,
	reason string,
	message string,
) bool {
	existing := meta.FindStatusCondition(status.Conditions, conditionType)

	if existing != nil &&
		existing.Status == conditionStatus &&
		existing.Reason == reason &&
		existing.Message == message {
		return false
	}

	meta.SetStatusCondition(&status.Conditions, metav1.Condition{
		Type:    conditionType,
		Status:  conditionStatus,
		Reason:  reason,
		Message: message,
	})
	return true
}
