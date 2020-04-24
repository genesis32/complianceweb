package server

// A list of permissions the system supports.
const (
	UserCreatePermission               = "user.create.execute"
	UserUpdatePermission               = "user.update.execute"
	UserReadPermission                 = "user.read.execute"
	OrganizationRolesAssignPermission  = "organization.roles.assign.execute"
	OrganizationCreatePermission       = "organization.create.execute"
	SystemOrganizationCreatePermission = "system.organization.create.execute"
	SystemUserCreatePermission         = "system.user.create.execute"
)
