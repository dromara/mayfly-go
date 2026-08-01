import { defineStore } from 'pinia';
import { milvusApi } from '@/views/ops/milvus/api';
import { setCurrentAcName } from './authCert';

/**
 * Milvus 动态 store 工厂函数，支持按 tabKey 隔离状态。
 * - 不传 id 或传 'milvusStore' 时返回全局默认 store（向后兼容）
 * - 传自定义 id（如 tabKey）时返回独立的 per-tab store
 */
export const useMilvusStore = (id: string = 'milvusStore') =>
    defineStore(id, {
        state: (): MilvusState => ({
            dbs: [],
            selectedDb: 'default',
            collections: [],
            selectedCollection: '',
            authCertName: '',
        }),
        actions: {
            setDbs(dbs: MilvusDb[]) {
                this.dbs = dbs;
            },
            async refreshDbs(milvusId: number) {
                const res = await milvusApi.listDatabases(milvusId);
                // res 通过dbid排序
                res.sort((a: import('@/views/ops/milvus/types').IDatabase, b: import('@/views/ops/milvus/types').IDatabase) => {
                    return (a.create_time || '').localeCompare(b.create_time || '');
                });
                this.dbs = res;
                if (res.length > 0) {
                    this.selectedDb = res[0].name;
                    milvusApi.useDatabase(Number(res[0].id), res[0].name);
                }
            },
            setSelectedDb(db: string) {
                this.selectedDb = db;
            },
            setSelectedCollection(coll: string) {
                this.selectedCollection = coll;
            },
            setCollections(collections: any[]) {
                collections.sort();
                this.collections = collections;
                // 默认选中第一个 collection
                if (!this.selectedCollection && this.collections.length > 0) {
                    this.setSelectedCollection((this.collections[0] as any)?.name || this.collections[0]);
                }
            },
            clear() {
                this.collections = [];
                this.selectedCollection = '';
                this.selectedDb = 'default';
                this.dbs = [];
            },
            setAuthCertName(name: string) {
                this.authCertName = name;
                setCurrentAcName(name);
            },
        },
    })();
