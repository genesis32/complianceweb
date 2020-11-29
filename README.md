# [Name TBD] Compliance Web

## What is this?

The goal of this is to be a very lightweight deployable service used to make access control decisions against a hierarchy. You create users and add them to a hierarchy of organizations and their resources, users can then perform actions only against those resources in the organizations they have visibility over.

I had some functionality in an earlier version that included a capability to add custom resources you could manage through the service. I will be adding that back.
[See an example](https://github.com/genesis32/complianceweb/tree/aws_gcp_resources/resources)

## Dependencies
* Postgresql 12 (needed for jsonb) w/ ltree extension
    * ltree extension: `create extension ltree` 
* User login is only available through auth0 ATM
    
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
    
Visit [the login page](http://localhost:3000/webapp), click LogIn, and ensure you get back an auth0 jwt at the end of the flow. You can use this jwt to make API calls against the services.

## TODO 

* Add back in an example resource type using the new model
* Audit logging