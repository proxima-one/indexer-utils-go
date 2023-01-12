# Utils for Proxima indexing services

## Logger

To create a Logger you just need to pass `io.Writer` to write to. Typically, it is `os.Stdout`.
   
```go
logger := utils.NewLogger(os.Stdout)
```

Now you need to register streams. There are two ways to register a stream:

- If your stream has defined start and end points, you can just use: 
  ```go
  logger.UpdateStream(streamId, startOffset, endOffset)
  ```
- Another option is made for live streams:
  ```go
  go logger.StartLiveStreamUpdate(ctx, streamId, startOffset, registry.FindStream, time.Hour)
  ```
  This will update stream metadata according to the passed interval. </br>
  You can use `proximaclient.StreamRegistryClient` from [streamdb-client-go](https://github.com/proxima-one/streamdb-client-go) package as a `registry`.

To start logging you need to pass it log interval:
```go
go logger.StartLogging(ctx, 10*time.Second)
```

And pass it every processed event:
```go
logger.EventProcessed(streamId, event)
```

Now it is automatically writing the following tables:
```
╭───────────┬───────────┬─────────────────────┬─────┬───────────┬───────┬───────────┬───────────╮
│ STREAM ID │    HEIGHT │ CURRENT TIMESTAMP   │ LAG │ AVG SPEED │ SPEED │ PROCESSED │ REMAINING │
├───────────┼───────────┼─────────────────────┼─────┼───────────┼───────┼───────────┼───────────┤
│ stream.id │ 149977831 │ 2022-12-29 12:15:11 │ 3s  │ 2.70      │ 0.70  │ 100.00%   │ live      │
╰───────────┴───────────┴─────────────────────┴─────┴───────────┴───────┴───────────┴───────────╯
```
Here, the AVG SPEED is an <b>average</b> speed and being calculated as `eventsProcessedSinceStart / timeSinceStart`.

SPEED is "instant speed" - <b>current</b> consumer speed. It is calculated as `eventsSinceLastLog / timeSinceLastLog`.

<i>Every `logger` gorotine stops as its context is closed.</i>   
