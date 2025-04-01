# SQL-PROXY

## About 

A lightweight REST service designed to replace ADODB calls in legacy software systems that support web requests. This service streamlines database interactions while maintaining security and efficiency.

For example, you can remove all Linux-incompatible components, such as the following items: 
```
Connection = New COMObject("ADODB.Connection");
Connection.Open(ConnectionString);
```
and use web requests in a similar manner instead, using a simple library to receive SQL query results in a JSON with variable columns you defined
in SQL select statement:
```
Function OpenSQLConnection(ConnectionString) Export
  HTTP = New HTTPConnection;
  Path = "/api/v1/connection";
  ...
```


## Key features:

* Multi-Database Support : Compatible with PostgreSQL, Microsoft SQL Server, and MySQL databases. Additional standard Golang database drivers can be integrated as needed;
* Secure Credential Management : Does not store SQL credentials, ensuring sensitive information remains protected;
* Secure Communication : Supports HTTPS for secure data transmission;
* Efficient Connection Pooling : Utilizes a shared, reusable SQL connection pool with automated maintenance tasks to remove stale or dead connections;
* Command Support : Currently supports all SQL commands with no limitation. The SELECT command returns query results as a flexible JSON-formatted recordset;
* Result Limitation : Allows configuration to limit the number of rows returned by SELECT statements;
* Prepared Statements : Implementation of prepared statements is planned for future development;
* Flexible Binding : Can bind to localhost or any specified IP address for enhanced security. By default, it is intended to bind to localhost and run alongside legacy software;
* Security Responsibility : Does not perform SQL query validation for security purposes. It is the responsibility of the user to configure appropriate database privileges.
* Monitoring and Metrics : Provides Prometheus metrics for performance monitoring and observability; 

## API description

See Swagger OpenAPI 3.0 specification in /src/docs/api

## How to compile

```
make prod
```

## How to run

Settings are passed with environment variables, see Makefile for details:

```
BIND_ADDR=localhost BIND_PORT=8081 MAX_ROWS=10000 LOG_LEVEL=3 sql-proxy
```
