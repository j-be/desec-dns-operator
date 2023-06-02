package util

import (
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
)

const TOKEN = "<token>"

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
