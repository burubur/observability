# Observability

## Why needed?

To observe how's the system really works

To watch this components:

- process
- CPU Usage
- Memory Usage
- Disk I/O Operation
- File Descriptor Usage
- Goroutine Numbers

## How to start

### 1. Running the microservice
#### 1a. Create docker image for the microservice
#### 1b. Spin up the service

### 2. Running the instrumentation tools
#### 1. metric exporter -> collecting metric data
#### 1. Spin up storage -> storing metric data
#### 2. http server -> serving metric data

#### 2b. Spin up the logger agent, storage and visualization tools
#### 2c. Spin up the tracer, storage and visualization tools

### 3. Simulating traffic
#### 3a. Goroutine Overflow
#### 3a. Goroutine Optimization

### 4. Creating Metric Dashboard
#### 4a. North star dashboard (company level, division level)
#### 4b. Component dashboard (team level, app level)

## Tips

### What things that should go to metrics
#### Errors/Failures

### What things that should go to logger
#### Error Message
#### Error Detail

### What things that should go to tracer
#### Bottleneck
#### Latency Optimization