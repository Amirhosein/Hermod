# Hermod: Mythic, Mighty, Simple Message Broker 

Hermod is a fast and simple gRPC based broker. It's a mini practical project to sail through the Golang and most common used tools and frameworks including:

- gRPC/Protobuf
- Concurrent programming
- Storing data using Cassandra/Postrgres/Redis
- Load Testing using K6 and simple golang client
- Unit Testing
- Monitoring using Prometheus and Grafana
- Tracing using Jaeger
- Rate Limiting using envoy
- Caching and batching for highly better performance
- Deploying using docker and kubernetes
- Creating helm chart for easy deployment

<p align="center">
  <a href="https://skillicons.dev">
    <img src="https://skillicons.dev/icons?i=go,prometheus,grafana,postgres,cassandra,redis,kubernetes,docker" />
  </a>
</p>

# Structure
![overall architecture](https://s24.picofile.com/file/8453067350/Screenshot_2022_09_12_122319.png)
For better understanding the core and logical concept of Hermod, i've created a simple prezi presentation. You can find it [here](https://prezi.com/view/qqAU2Fd7MCXcTl3sJXxv/). Make sure to check it out!

## RPCs Description
- Publish Requst
```protobuf
message PublishRequest {
  string subject = 1;
  bytes body = 2;
  int32 expirationSeconds = 3;
}
```
- Fetch Request
```protobuf
message FetchRequest {
  string subject = 1;
  int32 id = 2;
}
```
- Subscribe Request
```protobuf
message SubscribeRequest {
  string subject = 1;
}
```
- RPC Service
```protobuf
service Broker {
  rpc Publish (PublishRequest) returns (PublishResponse);
  rpc Subscribe(SubscribeRequest) returns (stream MessageResponse);
  rpc Fetch(FetchRequest) returns (MessageResponse);
}
```

# Up and Running
## Docker 
```shell
docker-compose -f deployments/docker-compose.yml up
```
## Kubernetes
```shell
cd hermod
helm install hermod
```
