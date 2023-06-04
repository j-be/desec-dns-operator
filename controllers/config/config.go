package config

import (
	"os"

	"k8s.io/apimachinery/pkg/types"
)

type Config struct {
	Domain    string
	Namespace string
}

func NewConfigFor(configDir string) (Config, error) {
	domain, err := os.ReadFile(configDir + "/config/domain")
	if err != nil {
		return Config{}, err
	}
	namespace, err := os.ReadFile(configDir + "/config/namespace")
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
