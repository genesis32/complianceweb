package aws

type IAMUserCreateRequest struct {
	UserName string `validate:"min=4,max=16,regexp=^[A-Za-z0-9]*$"`
}
