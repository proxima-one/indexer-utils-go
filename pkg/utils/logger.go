package utils

import (
	"context"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/proxima-one/streamdb-client-go/pkg/proximaclient"

	"os"
	"time"
)

type Logger struct {
	file           *os.File
	findStreamFunc func(stream string) (*proximaclient.Stream, error)

	streamEventsToProcess chan streamEvent
}

type streamEvent struct {
	streamId string
	event    proximaclient.StreamEvent
}

func NewLogger(file *os.File, findStreamFunc func(stream string) (*proximaclient.Stream, error)) *Logger {
	return &Logger{
		file:                  file,
		findStreamFunc:        findStreamFunc,
		streamEventsToProcess: make(chan streamEvent, 10),
	}
}

func (logger *Logger) StartLogging(ctx context.Context, logInterval, streamMetadataUpdateInterval time.Duration) {
	go func() {
		type streamData struct {
			messagesProcessed               int64
			messagesProcessedWhenLastLogged int64
			lastProcessedEvent              *proximaclient.StreamEvent
			lastOffset                      *proximaclient.Offset
			startTime                       time.Time
		}
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
					avgSpeed := (1000 * data.messagesProcessed) / time.Now().Sub(data.startTime).Milliseconds()
					processedPercent := ""
					remainingTime := ""
					if data.lastOffset != nil {
						if data.lastProcessedEvent.Offset.Height >= data.lastOffset.Height {
							processedPercent = "100.00%"
							remainingTime = "live"
						} else {
							processedPercent = fmt.Sprintf("%.2f%%",
								float32(data.lastProcessedEvent.Offset.Height)/float32(data.lastOffset.Height)*100.,
							)
							remainingTime = time.Duration(
								int64(time.Second) * (data.lastOffset.Height - data.lastProcessedEvent.Offset.Height) / avgSpeed,
							).Truncate(time.Second).String()
						}
					}
					t.AppendRow(table.Row{
						streamId,
						data.lastProcessedEvent.Offset.Height,
						data.lastProcessedEvent.Timestamp.Time().Format("2006-01-02 15:04:05"),
						avgSpeed,
						1000 * (data.messagesProcessed - data.messagesProcessedWhenLastLogged) / lastLoggedTime.Sub(data.startTime).Milliseconds(),
						processedPercent,
						remainingTime,
					})
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
