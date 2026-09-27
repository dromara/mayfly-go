import { registerDbSqlExecProgress } from './db-sql-exec-progress';
import { registerDbDataImportProgress } from './db-data-import-progress';

export function initDbSysMsgs() {
    registerDbSqlExecProgress();
    registerDbDataImportProgress();
}
