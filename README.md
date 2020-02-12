# Enterprise Portal 2

## What is this?

It's a reimagining of Enterprise Portal 1. It separates resources from the organization 
tree and enables them to be managed separately. Specifically it allows organization
administrators and users authenticated through Auth0 to create organization trees, 
inside these trees they can put users, resources, and metadata.

## Dependencies
* Postgresql 12 (needed for jsonb) w/ ltree extension
    * ltree extension: `create extension ltree` 
* Google Cloud "Cloud Resource Manager" API is enabled for the project you want to manage.
* A gcp service account w/ "Service Account Creator" and "Service Account Key Creator" role.
    
## Running Locally

Preload the database with data:

    psql < schema/00schema.sql
    psql < schema/01seed.sql
    psql < schema/02resource_schema.sql

To download dependencies and build the executable:

    go get -u ./... && go build

To run the tests and make sure everything is sane:

    cd integration_tests && GIN_MODE=release dotenv ../test.env go test  -v

Start the app:

    go build && dotenv test.env ./complianceweb

Create a System Admin Account

```shell script
curl -X POST "http://localhost:3000/system/bootstrap" -H "Content-Type: application/json" --data '{"SystemAdminName":"sysadmin0"}'
```
    
Visit [the login page](http://localhost:3000/webapp), click LogIn,
and ensure you get back an auth0 jwt at the end of the flow. You can use this jwt
to make API calls against the services.

### Docker Notes

make all:

    make app && make database

Run the database:

    docker run --rm --name enterpriseportal2-postgres -p 9876:5432 hilobit:enterpriseportal2-db
    
Run the app:

    docker run --link enterpriseportal2-postgres -p 3000:8080 hilobit:enterpriseportal2

## Resource Types

### GCP

TODO 
