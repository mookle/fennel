# 0006 - Event broker: RabbitMQ

Status: Accepted

## Context

ADR-0005 introduces an asynchronous event emitted by `order` and consumed by `product`.

We need a broker that both Go and Elixir support well. It must give durable, acknowledged delivery, and it must be simple to run in the cloud and within a development cluster.

RabbitMQ has mature clients for both Elixir and Go. A durable quorum queue plus publisher acks provide an at-least-once guarantee. Simple

NATS JetStream is lighter and has a good Go client, but the Elixir ecosystem around it is weaker. Kafka provides capabilities that are unnecessary for a single low-volume event in this vertical slice.

## Decision

Use RabbitMQ as the event broker between services.

## Consequences

- Services can communicate asynchronously through RabbitMQ rather than making direct service calls.
- The platform has a channel for future events to flow through. 
- RabbitMQ becomes a stateful infrastructure dependency.
- If event volume or retention requirements grow significantly, the broker choice may need revisiting.

