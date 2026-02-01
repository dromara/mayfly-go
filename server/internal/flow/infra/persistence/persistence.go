package persistence

import (
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newProcdefRepo())
	ioc.Register(newProcinstRepo())
	ioc.Register(newExectionRepo())
	ioc.Register(newProcinstTaskRepo())
	ioc.Register(newProcinstTaskCandidateRepo())
	ioc.Register(newHisProcinstOpRepo())
}

func GetProcinstTaskCandidateRepo() repository.ProcinstTaskCandidate {
	return ioc.Get[repository.ProcinstTaskCandidate]()
}
