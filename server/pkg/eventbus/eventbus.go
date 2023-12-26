package eventbus

import (
	"context"
	"errors"
	"mayfly-go/pkg/logx"
	"sync"
)

// BusSubscriber defines subscription-related bus behavior
type BusSubscriber interface {
	Subscribe(topic string, subId string, fn EventHandleFunc) error
	SubscribeAsync(topic string, subId string, fn EventHandleFunc, transactional bool) error
	SubscribeOnce(topic string, subId string, fn EventHandleFunc) error
	Unsubscribe(topic string, subId string) error
}

// BusPublisher defines publishing-related bus behavior
type BusPublisher interface {
	Publish(ctx context.Context, topic string, val any)
}

// BusController defines bus control behavior (checking handler's presence, synchronization)
type BusController interface {
	WaitAsync()
}

// Bus englobes global (subscribe, publish, control) bus behavior
type Bus interface {
	BusController
	BusSubscriber
	BusPublisher
}

// EventBus - box for handlers and callbacks.
type EventBus struct {
	subscriberManager map[string]*SubscriberManager // topic -> SubscriberManager
	lock              sync.Mutex                    // a lock for the map
	wg                sync.WaitGroup
}

func New() Bus {
	b := &EventBus{
		make(map[string]*SubscriberManager),
		sync.Mutex{},
		sync.WaitGroup{},
	}
	return Bus(b)
}

type Event struct {
	Topic string
	Val   any
}

// 订阅者们的事件处理器
type SubscriberManager struct {
	// 事件处理器 subId -> handler
	handlers map[string]*eventHandler
}

func NewSubscriberManager() *SubscriberManager {
	return &SubscriberManager{
		handlers: make(map[string]*eventHandler),
	}
}

func (sm *SubscriberManager) addSubscriber(subId string, eventHandler *eventHandler) {
	sm.handlers[subId] = eventHandler
}

func (sm *SubscriberManager) delSubscriber(subId string) {
	delete(sm.handlers, subId)
}

// 事件处理函数
type EventHandleFunc func(ctx context.Context, event *Event) error

type eventHandler struct {
	handlerFunc   EventHandleFunc // 事件处理函数
	once          bool            // 是否只执行一次
	async         bool
	transactional bool

	sync.Mutex // lock for an event handler - useful for running async callbacks serially
}

func (bus *EventBus) Subscribe(topic string, subId string, fn EventHandleFunc) error {
	eh := &eventHandler{}
	eh.handlerFunc = fn
	return bus.doSubscribe(topic, subId, eh)
}

// SubscribeAsync subscribes to a topic with an asynchronous callback
// Transactional determines whether subsequent callbacks for a topic are
func (bus *EventBus) SubscribeAsync(topic string, subId string, fn EventHandleFunc, transactional bool) error {
	eh := &eventHandler{}
	eh.handlerFunc = fn
	eh.async = true
	return bus.doSubscribe(topic, subId, eh)
}

// SubscribeOnce subscribes to a topic once. Handler will be removed after executing.
func (bus *EventBus) SubscribeOnce(topic string, subId string, fn EventHandleFunc) error {
	eh := &eventHandler{}
	eh.handlerFunc = fn
	eh.once = true
	return bus.doSubscribe(topic, subId, eh)
}

func (bus *EventBus) Unsubscribe(topic string, subId string) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	subManager := bus.subscriberManager[topic]
	if subManager == nil {
		return errors.New("该主题不存在订阅者")
	}
	subManager.delSubscriber(subId)
	return nil
}

func (bus *EventBus) Publish(ctx context.Context, topic string, val any) {
	bus.lock.Lock() // will unlock if handler is not found or always after setUpPublish
	defer bus.lock.Unlock()
	logx.Debugf("主题-[%s]-发布了事件", topic)
	event := &Event{
		Topic: topic,
		Val:   val,
	}
	subscriberManager := bus.subscriberManager[topic]
	if subscriberManager == nil {
		return
	}

	handlers := subscriberManager.handlers
	if len(handlers) == 0 {
		return
	}

	for subId, handler := range handlers {
		logx.Debugf("订阅者-[%s]-开始执行主题-[%s]-发布的事件", subId, topic)
		if handler.once {
			subscriberManager.delSubscriber(subId)
		}
		if !handler.async {
			bus.doPublish(ctx, handler, event)
		} else {
			bus.wg.Add(1)
			if handler.transactional {
				bus.lock.Unlock()
				handler.Lock()
				bus.lock.Lock()
			}
			go bus.doPublishAsync(ctx, handler, event)
		}
	}
}

// WaitAsync waits for all async callbacks to complete
func (bus *EventBus) WaitAsync() {
	bus.wg.Wait()
}

func (bus *EventBus) doSubscribe(topic string, subId string, handler *eventHandler) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	logx.Debugf("订阅者-[%s]-订阅了主题-[%s]", subId, topic)
	subManager := bus.subscriberManager[topic]
	if subManager == nil {
		subManager = NewSubscriberManager()
		bus.subscriberManager[topic] = subManager
	}
	subManager.addSubscriber(subId, handler)
	return nil
}

func (bus *EventBus) doPublish(ctx context.Context, handler *eventHandler, event *Event) error {
	err := handler.handlerFunc(ctx, event)
	if err != nil {
		logx.Errorf("订阅者执行主题[%s]失败: %s", event.Topic, err.Error())
	}
	return err
}

func (bus *EventBus) doPublishAsync(ctx context.Context, handler *eventHandler, event *Event) {
	defer bus.wg.Done()
	if handler.transactional {
		defer handler.Unlock()
	}
	bus.doPublish(ctx, handler, event)
}
