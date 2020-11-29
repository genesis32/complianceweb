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

make all:

    make app database

Run the database:

    docker run --rm --name enterpriseportal2-postgres -p 9876:5432 enterpriseportal2-db:latest
    
Run the app:

    docker run --env ENV=test --env PGSQL_CONNECTION_STRING="port=5432 host=enterpriseportal2-postgres user=ep2 password=ep2 dbname=enterpriseportal2 sslmode=disable" --link enterpriseportal2-postgres -p 3000:8080 enterpriseportal2:latest

To run the tests and make sure everything is sane:

    dotenv test.env go test -v ./...

Create a System Admin Account

```shell script
curl -X POST "http://localhost:3000/system/bootstrap" -H "Content-Type: application/json" --data '{"SystemAdminName":"sysadmin0"}'
```
    
Visit [the login page](http://localhost:3000/webapp), click LogIn,
and ensure you get back an auth0 jwt at the end of the flow. You can use this jwt
to make API calls against the services.

### Docker Notes


## Resource Types

### GCP

TODO 
