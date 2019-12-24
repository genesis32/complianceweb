INSERT INTO permission VALUES (1, 'service account create', 'serviceaccount.create.execute');
INSERT INTO permission VALUES (2, 'user create', 'user.create.execute');
INSERT INTO permission VALUES (3, 'organization create', 'organization.create.execute');

INSERT INTO role VALUES (1, 'Service Account Creator');
INSERT INTO role_permission_xref VALUES (1,1);

INSERT INTO role VALUES (2, 'Admin');
INSERT INTO role_permission_xref VALUES (2,1);
INSERT INTO role_permission_xref VALUES (2,2);
INSERT INTO role_permission_xref VALUES (2,3);

