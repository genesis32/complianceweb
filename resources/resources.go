package resources

var loadedResources = []OrganizationResourceAction{
	GcpServiceAccountResourcePostAction{},
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

type OperationParameters map[string]interface{}

type OrganizationResourceAction interface {
	Name() string
	InternalKey() string

	Method() string
	PermissionName() string
	Execute(params OperationParameters)
}
