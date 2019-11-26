package webhandlers

import (
	"bytes"
	"io"
	"mime/multipart"
)

type OrganizationForm struct {
	// TODO: Change me to a string
	ParentOrganizationID int64                 `form:"parent_organization_id"`
	Name                 string                `form:"orgname" binding:"required"`
	AccountCredential    *multipart.FileHeader `form:"master_account_json"`
}

type AddUserToOrganizationForm struct {
	Name           string `json:"name"`
	OrganizationId int64  `json:"organizationId"`
}

func (o *OrganizationForm) RetrieveContents() string {
	if o.AccountCredential != nil {
		buf := bytes.NewBuffer(nil)

		f, err := o.AccountCredential.Open()
		if err != nil {
			return ""
		}
		defer f.Close()
		io.Copy(buf, f)
		return string(buf.Bytes())
	}
	return ""
}
