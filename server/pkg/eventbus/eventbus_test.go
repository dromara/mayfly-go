package eventbus

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestSubscribe(t *testing.T) {
	bus := New()

	bus.SubscribeAsync("topic", "sub5", func(ctx context.Context, event *Event) error {
		time.Sleep(5 * time.Second)
		fmt.Printf("%s -> %s -> %d\n", "sub5", event.Topic, event.Val)
		return nil
	}, true)

	bus.SubscribeOnce("topic", "sub1", func(ctx context.Context, event *Event) error {
		fmt.Printf("%s -> %s -> %d\n", "sub1", event.Topic, event.Val)
		return nil
	})

	bus.Subscribe("topic", "sub2", func(ctx context.Context, event *Event) error {
		time.Sleep(5 * time.Second)
		return errors.New("失败。。。。")
	})

	bus.SubscribeAsync("topic", "sub3", func(ctx context.Context, event *Event) error {
		fmt.Printf("%s -> %s -> %d\n", "sub3", event.Topic, event.Val)
		return nil
	}, false)

	bus.SubscribeAsync("topic", "sub4", func(ctx context.Context, event *Event) error {
		time.Sleep(5 * time.Second)
		fmt.Printf("%s -> %s -> %d\n", "sub4", event.Topic, event.Val)
		return nil
	}, false)

	bus.Publish(context.Background(), "topic", 10)
	bus.Publish(context.Background(), "topic", 20)
	bus.WaitAsync()
}
