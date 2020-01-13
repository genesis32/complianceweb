package resources

type OperationParameters map[string]interface{}

type OrganizationResourceAction interface {
	Name() string
	InternalKey() string

	Method() string
	Allowed(permissions []string) bool
	PermissionName() string
	Execute(params OperationParameters)
}
