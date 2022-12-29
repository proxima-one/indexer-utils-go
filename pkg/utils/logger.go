package utils

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/proxima-one/streamdb-client-go/pkg/proximaclient"
	"golang.org/x/exp/constraints"
	"io"

	"time"
)

type Logger struct {
	file io.Writer

	streamEventsToProcess chan streamEvent
	streamUpdates         chan updateStreamRequest
}

func NewLogger(file io.Writer) *Logger {
	return &Logger{
		file:                  file,
		streamEventsToProcess: make(chan streamEvent, 10),
		streamUpdates:         make(chan updateStreamRequest, 1),
	}
}

type updateStreamRequest struct {
	StreamId    string
	FirstOffset proximaclient.Offset
	LastOffset  proximaclient.Offset
}

type streamEvent struct {
	streamId string
	event    proximaclient.StreamEvent
}

type streamData struct {
	messagesProcessed               int64
	messagesProcessedWhenLastLogged int64
	lastProcessedEvent              *proximaclient.StreamEvent
	firstOffset                     *proximaclient.Offset
	lastOffset                      *proximaclient.Offset
	startTime                       time.Time
}

func (logger *Logger) UpdateStream(streamId string, startOffset, endOffset proximaclient.Offset) {
	logger.streamUpdates <- updateStreamRequest{
		StreamId:    streamId,
		FirstOffset: startOffset,
		LastOffset:  endOffset,
	}
}

func (logger *Logger) StartLogging(ctx context.Context, logInterval time.Duration) {
	go func() {
		streamDataById := make(map[string]*streamData)

		t := table.NewWriter()
		t.SetStyle(table.StyleRounded)
		t.SetOutputMirror(logger.file)
		t.AppendHeader(table.Row{"Stream id", "Height", "Current Timestamp", "Lag", "Avg Speed", "Speed", "Processed", "Remaining"})

		log := time.NewTicker(logInterval)
		defer log.Stop()

		lastLoggedTime := time.Now()

		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return

			case req := <-logger.streamUpdates: // prioritize stream updates over other channels
				streamDataById[req.StreamId] = updateStreamData(streamDataById[req.StreamId], &req)

			default:
				break
			}

			select {
			case <-ctx.Done():
				return

			case <-log.C:
				if len(streamDataById) == 0 {
					continue
				}
				t.ResetRows()
				for streamId, data := range streamDataById {
					if data.firstOffset == nil || data.lastOffset == nil || data.lastProcessedEvent == nil {
						continue
					}
					t.AppendRow(streamRowFromData(lastLoggedTime, streamId, data))
					data.messagesProcessedWhenLastLogged = data.messagesProcessed
				}
				lastLoggedTime = time.Now()
				t.Render()

			case req := <-logger.streamUpdates:
				streamDataById[req.StreamId] = updateStreamData(streamDataById[req.StreamId], &req)

			case event := <-logger.streamEventsToProcess:
				data := streamDataById[event.streamId]
				if data != nil {
					data.lastProcessedEvent = &event.event
					data.messagesProcessed++
				}
			}
		}
	}()
}

func (logger *Logger) EventProcessed(streamId string, event proximaclient.StreamEvent) {
	logger.streamEventsToProcess <- streamEvent{
		streamId: streamId,
		event:    event,
	}
}

func (logger *Logger) StartLiveStreamUpdate(
	ctx context.Context,
	streamId string,
	startOffset proximaclient.Offset,
	findStream func(stream string) (*proximaclient.Stream, error),
	timeout time.Duration) {

	t := time.NewTicker(timeout)
	defer t.Stop()
	for ctx.Err() == nil {
		lastOffset := lastOffsetForStream(streamId, findStream)
		if lastOffset != nil {
			logger.UpdateStream(streamId, startOffset, *lastOffset)
		}

		select {
		case <-ctx.Done():
			break
		case <-t.C:
			break
		}
	}
}

func updateStreamData(data *streamData, req *updateStreamRequest) *streamData {
	res := data
	if res == nil {
		res = new(streamData)
		res.startTime = time.Now()
	}
	res.firstOffset = &req.FirstOffset
	res.lastOffset = &req.LastOffset
	return res
}

func divideAsFloats[T constraints.Integer](a, b T) float32 {
	return float32(a) / float32(b)
}

func calcProcessedPercent(lastProcessedOffset, firstOffset, lastOffset *proximaclient.Offset) string {
	if lastProcessedOffset.Height >= lastOffset.Height {
		return "100.00%"
	}
	return fmt.Sprintf("%.2f%%",
		100.*divideAsFloats(lastProcessedOffset.Height-firstOffset.Height, lastOffset.Height-firstOffset.Height),
	)
}

func calcRemainingTime(lastProcessedOffset, lastOffset *proximaclient.Offset, avgSpeed float32) string {
	if lastProcessedOffset.Height >= lastOffset.Height {
		return "live"
	}
	return time.Duration(
		float32(time.Second) * float32(lastOffset.Height-lastProcessedOffset.Height) / avgSpeed,
	).Truncate(time.Second).String()
}

func streamRowFromData(lastLoggedTime time.Time, streamId string, data *streamData) table.Row {
	avgSpeed := divideAsFloats(1000*data.messagesProcessed, time.Now().Sub(data.startTime).Milliseconds())
	processedPercent := calcProcessedPercent(&data.lastProcessedEvent.Offset, data.firstOffset, data.lastOffset)
	remainingTime := calcRemainingTime(&data.lastProcessedEvent.Offset, data.lastOffset, avgSpeed)
	return table.Row{
		streamId,
		data.lastProcessedEvent.Offset.Height,
		data.lastProcessedEvent.Timestamp.Time().Format("2006-01-02 15:04:05"),
		time.Now().Sub(data.lastProcessedEvent.Timestamp.Time()).Truncate(time.Second).String(),
		fmt.Sprintf("%.2f", avgSpeed),
		fmt.Sprintf("%.2f", divideAsFloats(
			1000.*(data.messagesProcessed-data.messagesProcessedWhenLastLogged),
			time.Now().Sub(lastLoggedTime).Milliseconds(),
		)),
		processedPercent,
		remainingTime,
	}
}

func lastOffsetForStream(streamId string, findStream func(stream string) (*proximaclient.Stream, error)) *proximaclient.Offset {
	meta, err := findStream(streamId)
	if err != nil {
		return nil
	}
	endpoints := meta.Endpoints
	var maxOffset *proximaclient.Offset
	for _, endpoint := range endpoints {
		if maxOffset == nil || endpoint.Stats.EndOffset.Height > maxOffset.Height {
			maxOffset = endpoint.Stats.EndOffset
		}
	}
	return maxOffset
}
