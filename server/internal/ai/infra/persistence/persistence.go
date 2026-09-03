package persistence

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.RegisterByType[*conversationRepoImpl]()
	ioc.RegisterByType[*turnItemRepoImpl]()
	ioc.RegisterByType[*memoryRepoImpl]()
	ioc.RegisterByType[*skillRepoImpl]()
	ioc.RegisterByType[*skillResourceRepoImpl]()
	ioc.RegisterByType[*mcpServerRepoImpl]()
}
