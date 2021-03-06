package project

var (
	description = "Organization operator manages namespaces based on Organization CR."
	gitSHA      = "n/a"
	name        = "organization-operator"
	source      = "https://github.com/giantswarm/organization-operator"
	version     = "0.9.1-dev"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
