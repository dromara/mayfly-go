package repository

type DbBackup interface {
	DbJob

	GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error)
}
