package utils

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/proxima-one/streamdb-client-go/pkg/proximaclient"
	"golang.org/x/exp/constraints"

	"os"
	"time"
)

type Logger struct {
	file           *os.File
	findStreamFunc func(stream string) (*proximaclient.Stream, error)

	streamEventsToProcess chan streamEvent
}

func NewLogger(file *os.File, findStreamFunc func(stream string) (*proximaclient.Stream, error)) *Logger {
	return &Logger{
		file:                  file,
		findStreamFunc:        findStreamFunc,
		streamEventsToProcess: make(chan streamEvent, 10),
	}
}

type streamEvent struct {
	streamId string
	event    proximaclient.StreamEvent
}

type streamData struct {
	messagesProcessed               int64
	messagesProcessedWhenLastLogged int64
	lastProcessedEvent              *proximaclient.StreamEvent
	lastOffset                      *proximaclient.Offset
	startTime                       time.Time
}

func (logger *Logger) StartLogging(ctx context.Context, logInterval, streamMetadataUpdateInterval time.Duration) {
	go func() {
		streamDataById := make(map[string]*streamData)

		t := table.NewWriter()
		t.SetStyle(table.StyleRounded)
		t.SetOutputMirror(logger.file)
		t.AppendHeader(table.Row{"Stream id", "Height", "Current Timestamp", "Avg Speed", "Speed", "Processed", "Remaining"})

		log := time.Tick(logInterval)
		updateMeta := time.Tick(streamMetadataUpdateInterval)

		lastLoggedTime := time.Now()

		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return

			case <-log:
				if len(streamDataById) == 0 {
					continue
				}
				t.ResetRows()
				for streamId, data := range streamDataById {
					t.AppendRow(streamRowFromData(lastLoggedTime, streamId, data))
					data.messagesProcessedWhenLastLogged = data.messagesProcessed
				}
				lastLoggedTime = time.Now()
				t.Render()

			case <-updateMeta:
				for streamId := range streamDataById {
					maxOffset := logger.streamMaxOffset(streamId)
					if maxOffset != nil {
						streamDataById[streamId].lastOffset = maxOffset
					}
				}

			case event := <-logger.streamEventsToProcess:
				data := streamDataById[event.streamId]
				if data == nil {
					data = new(streamData)
					streamDataById[event.streamId] = data
					data.lastOffset = logger.streamMaxOffset(event.streamId)
					data.startTime = time.Now()
				}

				data.lastProcessedEvent = &event.event
				data.messagesProcessed++
			}
		}
	}()
}

func divideAsFloats[T constraints.Integer](a, b T) float32 {
	return float32(a) / float32(b)
}

func calcProcessedPercent(lastProcessedOffset, lastOffset *proximaclient.Offset) string {
	if lastOffset == nil {
		return ""
	}
	if lastProcessedOffset.Height >= lastOffset.Height {
		return "100.00%"
	}
	return fmt.Sprintf("%.2f%%",
		100.*divideAsFloats(lastProcessedOffset.Height, lastOffset.Height),
	)
}

func calcRemainingTime(lastProcessedOffset, lastOffset *proximaclient.Offset, avgSpeed float32) string {
	if lastOffset == nil {
		return ""
	}
	if lastProcessedOffset.Height >= lastOffset.Height {
		return "live"
	}
	return time.Duration(
		float32(time.Second) * float32(lastOffset.Height-lastProcessedOffset.Height) / avgSpeed,
	).Truncate(time.Second).String()
}

func streamRowFromData(lastLoggedTime time.Time, streamId string, data *streamData) table.Row {
	avgSpeed := divideAsFloats(1000*data.messagesProcessed, time.Now().Sub(data.startTime).Milliseconds())
	processedPercent := calcProcessedPercent(&data.lastProcessedEvent.Offset, data.lastOffset)
	remainingTime := calcRemainingTime(&data.lastProcessedEvent.Offset, data.lastOffset, avgSpeed)
	return table.Row{
		streamId,
		data.lastProcessedEvent.Offset.Height,
		data.lastProcessedEvent.Timestamp.Time().Format("2006-01-02 15:04:05"),
		fmt.Sprintf("%.2f", avgSpeed),
		fmt.Sprintf("%.2f", divideAsFloats(
			1000.*data.messagesProcessed-data.messagesProcessedWhenLastLogged,
			lastLoggedTime.Sub(data.startTime).Milliseconds(),
		)),
		processedPercent,
		remainingTime,
	}
}

func (logger *Logger) EventProcessed(streamId string, event proximaclient.StreamEvent) {
	logger.streamEventsToProcess <- streamEvent{
		streamId: streamId,
		event:    event,
	}
}

func (logger *Logger) streamMaxOffset(streamId string) *proximaclient.Offset {
	meta, err := logger.findStreamFunc(streamId)
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
