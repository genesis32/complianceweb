INSERT INTO permission VALUES (1, 'service account create', 'serviceaccount.create.execute');
INSERT INTO permission VALUES (2, 'user create', 'user.create.execute');
INSERT INTO permission VALUES (3, 'organization create', 'organization.create.execute');
INSERT INTO permission VALUES (4, 'system update', 'system.update.execute');

INSERT INTO role VALUES (1, 'Service Account Creator');
INSERT INTO role_permission_xref VALUES (1,1);

INSERT INTO role VALUES (2, 'Organization Admin');
INSERT INTO role_permission_xref VALUES (2,1);
INSERT INTO role_permission_xref VALUES (2,2);
INSERT INTO role_permission_xref VALUES (2,3);

INSERT INTO role VALUES (3, 'System Admin');
INSERT INTO role_permission_xref VALUES (3,4);

-- GCP Service account roles and permissions
INSERT INTO permission VALUES (5, 'gcp service account create', 'gcp.serviceaccount.write.execute');

INSERT INTO role VALUES (4, 'GCP Administrator');
INSERT INTO role_permission_xref VALUES (5,4);

INSERT INTO registered_resources VALUES (1, 'GCP Service Accounts', 'gcp.serviceaccount', true);
