package repository

type DbRestore interface {
	DbJob

	GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error)
}
