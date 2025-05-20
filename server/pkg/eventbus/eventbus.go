package eventbus

import (
	"context"
	"errors"
	"mayfly-go/pkg/logx"
	"sync"
)

// BusSubscriber defines subscription-related bus behavior
type BusSubscriber[T any] interface {
	Subscribe(topic string, subId string, fn EventHandleFunc[T]) error
	SubscribeAsync(topic string, subId string, fn EventHandleFunc[T], transactional bool) error
	SubscribeOnce(topic string, subId string, fn EventHandleFunc[T]) error
	Unsubscribe(topic string, subId string) error
}

// BusPublisher defines publishing-related bus behavior
type BusPublisher[T any] interface {
	Publish(ctx context.Context, topic string, val T)

	PublishSync(ctx context.Context, topic string, val T) error
}

// BusController defines bus control behavior (checking handler's presence, synchronization)
type BusController interface {
	WaitAsync()
}

type Event[T any] struct {
	Topic string
	Val   T
}

// 订阅者们的事件处理器
type SubscriberManager[T any] struct {
	// 事件处理器 subId -> handler
	handlers map[string]*eventHandler[T]
}

func NewSubscriberManager[T any]() *SubscriberManager[T] {
	return &SubscriberManager[T]{
		handlers: make(map[string]*eventHandler[T]),
	}
}

func (sm *SubscriberManager[T]) addSubscriber(subId string, eventHandler *eventHandler[T]) {
	sm.handlers[subId] = eventHandler
}

func (sm *SubscriberManager[T]) delSubscriber(subId string) {
	delete(sm.handlers, subId)
}

// 事件处理函数
type EventHandleFunc[T any] func(ctx context.Context, event *Event[T]) error

type eventHandler[T any] struct {
	handlerFunc   EventHandleFunc[T] // 事件处理函数
	once          bool               // 是否只执行一次
	async         bool
	transactional bool

	sync.Mutex // lock for an event handler - useful for running async callbacks serially
}

// Bus englobes global (subscribe, publish, control) bus behavior
type Bus[T any] interface {
	BusController
	BusSubscriber[T]
	BusPublisher[T]
}

// EventBus - box for handlers and callbacks.
type EventBus[T any] struct {
	subscriberManager map[string]*SubscriberManager[T] // topic -> SubscriberManager
	lock              sync.Mutex                       // a lock for the map
	wg                sync.WaitGroup
}

func New[T any]() Bus[T] {
	b := &EventBus[T]{
		make(map[string]*SubscriberManager[T]),
		sync.Mutex{},
		sync.WaitGroup{},
	}
	return Bus[T](b)
}

func (bus *EventBus[T]) Subscribe(topic string, subId string, fn EventHandleFunc[T]) error {
	eh := &eventHandler[T]{}
	eh.handlerFunc = fn
	return bus.doSubscribe(topic, subId, eh)
}

// SubscribeAsync subscribes to a topic with an asynchronous callback
// Transactional determines whether subsequent callbacks for a topic are
func (bus *EventBus[T]) SubscribeAsync(topic string, subId string, fn EventHandleFunc[T], transactional bool) error {
	eh := &eventHandler[T]{}
	eh.handlerFunc = fn
	eh.async = true
	return bus.doSubscribe(topic, subId, eh)
}

// SubscribeOnce subscribes to a topic once. Handler will be removed after executing.
func (bus *EventBus[T]) SubscribeOnce(topic string, subId string, fn EventHandleFunc[T]) error {
	eh := &eventHandler[T]{}
	eh.handlerFunc = fn
	eh.once = true
	return bus.doSubscribe(topic, subId, eh)
}

func (bus *EventBus[T]) Unsubscribe(topic string, subId string) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	subManager := bus.subscriberManager[topic]
	if subManager == nil {
		return errors.New("there is no subscriber for this topic")
	}
	subManager.delSubscriber(subId)
	return nil
}

func (bus *EventBus[T]) Publish(ctx context.Context, topic string, val T) {
	bus.publishInternal(ctx, topic, val, false)
}

func (bus *EventBus[T]) PublishSync(ctx context.Context, topic string, val T) error {
	return bus.publishInternal(ctx, topic, val, true)
}

func (bus *EventBus[T]) publishInternal(ctx context.Context, topic string, val T, syncSubHandleErrorOnStop bool) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()

	event := &Event[T]{Topic: topic, Val: val}
	logx.Debugf("topic-[%s] - published the event", topic)

	subManager := bus.subscriberManager[topic]
	if subManager == nil || len(subManager.handlers) == 0 {
		return nil
	}

	var syncSubError error
	for subId, handler := range subManager.handlers {
		if handler.once {
			subManager.delSubscriber(subId)
		}

		// 同步执行处理
		if !handler.async {
			// 同步订阅者其中一个执行失败则停止后续订阅者处理字段设置为true，并且已经出现订阅者处理失败则跳过后续同步订阅者的处理
			if syncSubHandleErrorOnStop && syncSubError != nil {
				continue
			}

			logx.Debugf("subscriber-[%s] - starts executing events published by topic-[%s]", subId, topic)

			err := bus.doPublish(ctx, handler, event)
			if err != nil {
				syncSubError = err
				logx.Errorf("subscriber-[%s] failed to handle event topic-[%s]: %v", subId, topic, err)
			}
			continue
		}

		logx.Debugf("async subscriber-[%s] - starts executing events published by topic-[%s]", subId, topic)
		// 异步执行
		bus.wg.Add(1)
		if handler.transactional {
			bus.lock.Unlock()
			handler.Lock()
			bus.lock.Lock()
		}
		go bus.doPublishAsync(ctx, subId, handler, event)
	}

	return syncSubError
}

// WaitAsync waits for all async callbacks to complete
func (bus *EventBus[T]) WaitAsync() {
	bus.wg.Wait()
}

func (bus *EventBus[T]) doSubscribe(topic string, subId string, handler *eventHandler[T]) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	logx.Debugf("subscribers-[%s] -> subscribed to topic-[%s]", subId, topic)
	subManager := bus.subscriberManager[topic]
	if subManager == nil {
		subManager = NewSubscriberManager[T]()
		bus.subscriberManager[topic] = subManager
	}
	subManager.addSubscriber(subId, handler)
	return nil
}

func (bus *EventBus[T]) doPublish(ctx context.Context, handler *eventHandler[T], event *Event[T]) error {
	return handler.handlerFunc(ctx, event)
}

func (bus *EventBus[T]) doPublishAsync(ctx context.Context, subId string, handler *eventHandler[T], event *Event[T]) {
	defer bus.wg.Done()
	if handler.transactional {
		defer handler.Unlock()
	}
	if err := bus.doPublish(ctx, handler, event); err != nil {
		logx.Errorf("async subscriber-[%s] failed to execute topic-[%s]: %s", subId, event.Topic, err.Error())
	}
}
