module github.com/giantswarm/organization-operator

go 1.14

require (
	github.com/giantswarm/api-schema v0.7.1 // indirect
	github.com/giantswarm/apiextensions/v3 v3.26.0
	github.com/giantswarm/companyd-client-go v0.6.1
	github.com/giantswarm/credentiald/v2 v2.0.1
	github.com/giantswarm/exporterkit v0.2.1
	github.com/giantswarm/k8sclient/v5 v5.11.0
	github.com/giantswarm/k8smetadata v0.3.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/microkit v0.2.2
	github.com/giantswarm/micrologger v0.5.0
	github.com/giantswarm/operatorkit/v5 v5.0.0
	github.com/golang/mock v1.4.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/viper v1.7.1
	gopkg.in/resty.v1 v1.12.0
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)

replace (
	github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/etcd v3.3.13+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.13-gs
)
