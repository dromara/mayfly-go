package persistence

import (
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newProcdefRepo(), ioc.WithComponentName("ProcdefRepo"))
	ioc.Register(newProcinstRepo(), ioc.WithComponentName("ProcinstRepo"))
	ioc.Register(newExectionRepo(), ioc.WithComponentName("ExectionRepo"))
	ioc.Register(newProcinstTaskRepo(), ioc.WithComponentName("ProcinstTaskRepo"))
	ioc.Register(newProcinstTaskCandidateRepo(), ioc.WithComponentName("ProcinstTaskCandidateRepo"))
	ioc.Register(newHisProcinstOpRepo(), ioc.WithComponentName("HisProcinstTaskRepo"))
}

func GetProcinstTaskCandidateRepo() repository.ProcinstTaskCandidate {
	return ioc.Get[repository.ProcinstTaskCandidate]("ProcinstTaskCandidateRepo")
}
