INSERT INTO permission VALUES (1, 'service account create', 'serviceaccount.create.execute');
INSERT INTO permission VALUES (2, 'user create', 'user.create.execute');
INSERT INTO permission VALUES (3, 'organization create', 'organization.create.execute');
INSERT INTO permission VALUES (4, 'system update', 'system.update.execute');
INSERT INTO permission VALUES (5, 'gcp service account create', 'gcp.serviceaccount.write.execute');
INSERT INTO permission VALUES (6, 'system organization create', 'system.organization.create.execute');
INSERT INTO permission VALUES (7, 'user update', 'user.update.execute');
INSERT INTO permission VALUES (8, 'user update', 'user.read.execute');
INSERT INTO permission VALUES (9, 'system user create', 'system.user.create.execute');
INSERT INTO permission VALUES (10, 'gcp service account read', 'gcp.serviceaccount.read.execute');
INSERT INTO permission VALUES (11, 'aws create iam user', 'aws.iam.user.create.execute');

INSERT INTO role VALUES (2, 'Organization Admin');
INSERT INTO role_permission_xref VALUES (2,(SELECT id FROM permission WHERE value = 'serviceaccount.create.execute'));
INSERT INTO role_permission_xref VALUES (2,(SELECT id FROM permission WHERE value = 'user.create.execute'));
INSERT INTO role_permission_xref VALUES (2,(SELECT id FROM permission WHERE value = 'organization.create.execute'));
INSERT INTO role_permission_xref VALUES (2,(SELECT id FROM permission WHERE value = 'user.update.execute'));
INSERT INTO role_permission_xref VALUES (2,(SELECT id FROM permission WHERE value = 'user.read.execute'));

INSERT INTO role VALUES (3, 'System Admin');
INSERT INTO role_permission_xref VALUES (3,(SELECT id FROM permission WHERE value = 'system.update.execute'));
INSERT INTO role_permission_xref VALUES (3,(SELECT id FROM permission WHERE value = 'system.organization.create.execute'));
INSERT INTO role_permission_xref VALUES (3,(SELECT id FROM permission WHERE value = 'system.user.create.execute'));

INSERT INTO role VALUES (4, 'GCP Administrator');
INSERT INTO role_permission_xref VALUES (4,(SELECT id FROM permission WHERE value = 'gcp.serviceaccount.write.execute'));
INSERT INTO role_permission_xref VALUES (4,(SELECT id FROM permission WHERE value = 'gcp.serviceaccount.read.execute'));

INSERT INTO role VALUES (5, 'AWS Administrator');
INSERT INTO role_permission_xref VALUES (5,(SELECT id FROM permission WHERE value = 'aws.iam.user.create.execute'));


INSERT INTO registered_resources VALUES (1, 'GCP Service Accounts', 'gcp.serviceaccount', true);
INSERT INTO registered_resources VALUES (2, 'GCP Service Account Keys', 'gcp.serviceaccount.keys', true);
INSERT INTO registered_resources VALUES (3, 'AWS IAM User', 'aws.iam.user', true);

INSERT INTO settings (key, value) VALUES ('bootstrap.enabled', 'true');
INSERT INTO settings (key, value) VALUES ('oidc.issuer.baseurl', 'https://***REMOVED***.auth0.com/');
INSERT INTO settings (key, value) VALUES ('oidc.auth0.clientid', '***REMOVED***');
INSERT INTO settings (key, value) VALUES ('oidc.auth0.clientsecret', '***REMOVED***');
INSERT INTO settings (key, value) VALUES ('system.baseurl', 'http://localhost:3000');


