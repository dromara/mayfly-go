/**
 * 集合元信息域：统计 + 索引列表的加载与索引增删。
 *
 * 一次面板打开只发一跳（后端把统计与索引合并在 collectionMeta 里）：
 * 分开拉会出现「统计是刚才的、索引是两秒前的」这种半新半旧的画面。
 *
 * 写操作失败一律返回 false 并把错误留在 state：面板要就地说明为什么没改成，
 * 而不是一闪而过的 toast（错误文案已由请求层统一提示）。
 */
import { reactive } from 'vue';

import { i18n } from '@/i18n';
import { mongoApi } from '../../api';
import type { CollectionParam, MongoCollectionMeta } from '../../types';
import { parseIndexSpecs } from '../../docview/json';

interface MetaState {
    loading: boolean;
    /** 索引写入进行中标记（创建与删除共用：同一面板里不会并发） */
    writing: boolean;
    error: string;
    meta: MongoCollectionMeta | null;
}

export function useCollectionMeta() {
    const state = reactive<MetaState>({
        loading: false,
        writing: false,
        error: '',
        meta: null,
    });

    async function load(target: CollectionParam) {
        state.loading = true;
        state.error = '';
        try {
            state.meta = await mongoApi.collectionMeta.request(target);
            return true;
        } catch (e) {
            state.error = e instanceof Error ? e.message : '';
            return false;
        } finally {
            state.loading = false;
        }
    }

    /** 创建索引：文本先过形状校验，服务端仍会再校验一次（这里是省掉一次注定失败的往返） */
    async function createIndexes(target: CollectionParam, specsText: string) {
        const parsed = parseIndexSpecs(specsText);
        if (!parsed.ok) {
            state.error = i18n.global.t(parsed.issue.key);
            return false;
        }

        state.writing = true;
        state.error = '';
        try {
            await mongoApi.createIndexes.request({ ...target, indexes: parsed.value });
            return await load(target);
        } catch (e) {
            state.error = e instanceof Error ? e.message : '';
            return false;
        } finally {
            state.writing = false;
        }
    }

    /**
     * 删除索引。
     *
     * confirmName 与索引名同值：删索引会让依赖它的查询退化全集合扫描，服务端要求把名字再打一遍，
     * 前端把用户输入的原文传回去，不替用户「自动填上」——那就等于没有这道闸。
     */
    async function dropIndex(target: CollectionParam, index: string, confirmName: string) {
        state.writing = true;
        state.error = '';
        try {
            await mongoApi.dropIndex.request({ ...target, index, confirmName });
            return await load(target);
        } catch (e) {
            state.error = e instanceof Error ? e.message : '';
            return false;
        } finally {
            state.writing = false;
        }
    }

    function reset() {
        state.meta = null;
        state.error = '';
    }

    return { state, load, createIndexes, dropIndex, reset };
}
