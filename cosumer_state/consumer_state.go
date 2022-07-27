package model

import (
	"fmt"
	"time"
)

type ConsumerState struct {
	Id                   string    `bson:"current_message_id"`
	StreamId             string    `bson:"_id"`
	TotalMessages        int       `bson:"total_messages"`
	LastMessageTimestamp time.Time `bson:"last_message_timestamp"`
}

func (*ConsumerState) GetCollectionName() string {
	return "consumer_states"
}

func (c *ConsumerState) GetMessageToLog() string {
	return fmt.Sprint(
		"Stream:", c.StreamId,
		"Total messages:", c.TotalMessages,
		"Last message's id:", c.Id,
		"Last message's time:", c.LastMessageTimestamp,
	)
}
