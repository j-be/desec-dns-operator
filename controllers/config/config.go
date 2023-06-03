package config

import (
	"os"

	"k8s.io/apimachinery/pkg/types"
)

type Config struct {
	Domain    string
	Namespace string
}

func NewConfigFor() (Config, error) {
	domain, err := os.ReadFile("./mnt/config/domain")
	if err != nil {
		return Config{}, err
	}
	namespace, err := os.ReadFile("./mnt/config/namespace")
	if err != nil {
		return Config{}, err
	}

	return Config{
		Domain:    string(domain),
		Namespace: string(namespace),
	}, nil
}

func (d Config) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{Name: d.Domain, Namespace: d.Namespace}
}
