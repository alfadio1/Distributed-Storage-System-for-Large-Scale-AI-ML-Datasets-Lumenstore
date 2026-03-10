# LumenStore (Day 1 Architecture)

## Goal
Build a mini distributed object store inspired by GFS/S3, optimized for high-throughput sequential reads (ML/AI workloads), with replication and failure recovery added in later days.

## Components
### Master (Metadata Service)
- Tracks registered storage nodes (node_id -> address)
- Receives heartbeats to detect node liveness
- (Later) stores object -> chunk mapping and replication placements with a WAL

### Storage Node (Chunk Server)
- Runs a gRPC server (Health endpoint today)
- Registers itself with the Master on startup
- Sends periodic heartbeats

### Client
- Not implemented Day 1
- I (Later) will chunk files and upload/download in parallel for throughput

## Control Plane vs Data Plane
- Control plane: Master metadata + node membership/health
- Data plane: Storage nodes serve chunk reads/writes (later days)

## Failure Model (Day 1)
- Nodes are considered alive if they heartbeat periodically.
- (Later) Master will mark nodes dead after heartbeat timeout and trigger re-replication.

## What’s Implemented in Day 1
- gRPC contracts for MasterService and NodeService
- Master: RegisterNode + Heartbeat
- Node: Health + auto RegisterNode + heartbeat loop
- Docker Compose: master + 3 nodes running in separate containers