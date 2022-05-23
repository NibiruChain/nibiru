## Installation

1. Influx installation

We can install influx and grafana with brew:
```bash
brew update
brew install influxdb
brew install grafana
```

Once installed you can start the services with: 
```bash
brew services start grafana # Will start on 3000
brew services start influxdb # will start on 8086
```

## Configuration of influx and grafana

Once the services started, go in `http://localhost:8086/` and configure a new simple server with credentials.


You can then access grafana here:
`http://localhost:3000/login`
Just use admin for login and password.

Go to settings, data sources to add a new influxdb database.

Use the URL http://localhost:8086/ for database, and configure the credentials on the bottom using the info from grafana.


