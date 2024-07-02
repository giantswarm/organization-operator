package service

import (
	"time"
)

type Config struct {
	Kubernetes   KubernetesConfig
	ResyncPeriod time.Duration
}

type KubernetesConfig struct {
	InCluster      bool
	KubeConfigPath string
}

type Service struct {
	Config
}
