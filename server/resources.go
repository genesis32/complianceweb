package server

import (
	"github.com/genesis32/complianceweb/resources"
	"github.com/genesis32/complianceweb/resources/aws"
	"github.com/genesis32/complianceweb/resources/gcp"
)

var loadedResources = []resources.OrganizationResourceAction{
	&gcp.ServiceAccountResourcePostAction{},
	&gcp.ServiceAccountResourceKeyPostAction{},
	&gcp.ServiceAccountResourceListGetAction{},
	&gcp.ServiceAccountResourceKeyGetAction{},
	&aws.IAMUserCreateResourcePostAction{},
	&aws.IAMUserApproveResourcePostAction{},
}
