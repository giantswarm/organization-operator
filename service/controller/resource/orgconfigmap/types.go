package orgconfigmap

type configMapSpec struct {
	Name      string
	Namespace string
	Values    map[string]interface{}
}
