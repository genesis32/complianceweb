package resources

var loadedResources = []OrganizationResourceAction{
	GcpServiceAccountResourcePostAction{},
	GcpServiceAccountResourceGetAction{},
}

type OperationParameters map[string]interface{}

type OrganizationResourceAction interface {
	Name() string
	InternalKey() string

	Method() string
	Allowed(permissions []string) bool
	PermissionName() string
	Execute(params OperationParameters)
}

func FindResourceActions(internalKey string) []OrganizationResourceAction {
	var ret []OrganizationResourceAction
	for _, v := range loadedResources {
		if internalKey == v.InternalKey() {
			ret = append(ret, v)
		}
	}
	return ret
}
