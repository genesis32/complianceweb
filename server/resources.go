package server

import (
	"github.com/genesis32/complianceweb/resources"
	"github.com/genesis32/complianceweb/resources/gcp"
)

var loadedResources = []resources.OrganizationResourceAction{
	&gcp.GcpServiceAccountResourcePostAction{},
	&gcp.GcpServiceAccountResourceKeyPostAction{},
	&gcp.GcpServiceAccountResourceListGetAction{},
	&gcp.GcpServiceAccountResourceKeyGetAction{},
}

func FindResourceActions(internalKey string) []resources.OrganizationResourceAction {
	var ret []resources.OrganizationResourceAction
	for _, v := range loadedResources {
		if internalKey == v.InternalKey() {
			ret = append(ret, v)
		}
	}
	return ret
}
