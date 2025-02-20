# SQL-PROXY

## About 

A simple REST service to replace ADODB calls in any legacy software which supports web calls

## Key features:

* Supports PostgreSQL, Microsoft SQL and MySQL databases. Any other standard Golang drivers may be embedded if need;
* Does not store any SQL credentials;
* Supports secure HTTPS;
* Has common reused SQL connecton pool, and maintenance tasks to remove dead connections;
* Currently supports both 'select' and 'execute' commands. 'Select' returns the recordset in JSON;
* Rows to return in 'select' statements may be limited by settings;
* Prepared statements are in "to do" state;
* May bind to localhost or any other IP address for security. Primarily it is intended to bind to localhost and run alongside with the legacy software;
* Does not check SQL query for any security reasons, this must be done by setting user privileges.
* Provides Prometheus metrics; 

## API description

[coming soon]