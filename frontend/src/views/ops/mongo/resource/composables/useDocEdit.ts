/**
 * 文档编辑域：新增/编辑弹窗状态、JSON 校验与写回。
 *
 * 与查询域分开：这里只管「一份文档文本如何变成一次写入」，不关心结果列表怎么取。
 * 写回一律带 idToken + baseHash：前者定位精确类型的主键，后者让并发编辑显式冲突而不是静默覆盖。
 */
import { computed, reactive } from 'vue';

import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { i18n } from '@/i18n';
import { mongoApi } from '../../api';
import type { MongoDoc } from '../../types';
import { collectTypedFields, parseDocText, prettyDoc, typedFieldsSummary } from '../../docview/extjson';
import type { DocQueryTab } from './useDocQuery';

/** 编辑意图：新增不带定位信息，编辑必须带回读取时的令牌与指纹 */
export type DocEditIntent = 'add' | 'edit';

/** 编辑域状态形状（显式标注而不是内联断言，保住 intent 的联合类型推导） */
interface DocEditState {
    visible: boolean;
    intent: DocEditIntent;
    /** 编辑器内容，始终以文本为本源（含 $ 类型包装时就是它将被保存的内容） */
    text: string;
    /** 被编辑文档（新增时为 null） */
    target: MongoDoc | null;
    saving: boolean;
    /** 就地显示的错误（已翻译） */
    error: string;
}

export function useDocEdit(onChanged: () => Promise<boolean> | void) {
    const state = reactive<DocEditState>({
        visible: false,
        intent: 'add',
        text: '',
        target: null,
        saving: false,
        error: '',
    });

    /**
     * 当前文本里的 BSON 类型标注摘要，如 `Date ×1、ObjectID ×1`。
     *
     * 显示它是为了让人知道 `$date` 这类包装不是冗余：删掉包装等于把字段类型改掉，
     * 这是本次重构要消灭的损坏形态，所以必须让它可见而不是猜。
     */
    const typedSummary = computed(() => {
        const doc = parseDocText(state.text);
        return doc ? typedFieldsSummary(collectTypedFields(doc)) : '';
    });

    /**
     * 打开新增。
     *
     * 以当前页第一个文档为模板（同集合文档形状通常一致，从空白写更易出错），
     * 并去掉 _id 让服务端生成主键。
     */
    function openAdd(template?: MongoDoc) {
        state.intent = 'add';
        state.target = null;
        state.error = '';
        const doc = template?.doc ? { ...template.doc } : {};
        delete doc._id;
        state.text = JSON.stringify(doc, null, 4);
        state.visible = true;
    }

    function openEdit(doc: MongoDoc) {
        state.intent = 'edit';
        state.target = doc;
        state.error = '';
        state.text = prettyDoc(doc.doc);
        state.visible = true;
    }

    function close() {
        state.visible = false;
        state.error = '';
    }

    /** 校验编辑器内容：必须是 JSON 对象 */
    function parseEditorDoc(): Record<string, unknown> | null {
        const value = parseDocText(state.text);
        if (value === null || typeof value !== 'object' || Array.isArray(value)) {
            return null;
        }
        return value as Record<string, unknown>;
    }

    /**
     * 保存。
     *
     * 失败时保持弹窗打开：内容已经写了一半的用户操作不该因为一次网络错误全部丢失。
     * 返回是否真的写入了内容（noChange 也视为成功但不刷新，避免无意义重查）。
     */
    async function save(tab: DocQueryTab): Promise<boolean> {
        const doc = parseEditorDoc();
        if (!doc) {
            state.error = i18n.global.t('mongo.docMustBeObject');
            return false;
        }

        state.saving = true;
        try {
            if (state.intent === 'add') {
                const res = await mongoApi.insertDocs.request({
                    id: tab.mongoId,
                    database: tab.database,
                    collection: tab.collection,
                    docs: doc,
                });
                if (!res.insertedCount) {
                    return false;
                }
                Msg.success('mongo.insertSuccess');
                close();
                await onChanged();
                return true;
            }

            const target = state.target;
            if (!target?.idToken) {
                // 投影排除了 _id 的文档只能查看：没有可定位的主键，硬拼条件会打到别的文档上
                state.error = i18n.global.t('mongo.idTokenMissing');
                return false;
            }

            const res = await mongoApi.updateDoc.request({
                id: tab.mongoId,
                database: tab.database,
                collection: tab.collection,
                idToken: target.idToken,
                baseHash: target.hash,
                doc,
            });
            if (res.noChange) {
                Msg.info('mongo.docNoChange');
            } else {
                Msg.saveSuccess();
            }
            close();
            if (!res.noChange) {
                await onChanged();
            }
            return true;
        } catch (e) {
            // 请求层已 toast 后端文案（权限不足、乐观锁冲突、命令失败都各有说法），
            // 这里只把同一句话留在弹窗里，用户不必去回忆一闪而过的提示
            state.error = e instanceof Error ? e.message : '';
            return false;
        } finally {
            state.saving = false;
        }
    }

    /** 删除单个文档。deletedCount 可能为 0（他人已删），如实说明而不是假装成功 */
    async function remove(tab: DocQueryTab, doc: MongoDoc) {
        if (!doc.idToken) {
            Msg.error('mongo.idTokenMissing');
            return;
        }
        if (!(await useI18nDeleteConfirm(`${tab.database}.${tab.collection}`))) {
            // 取消或关掉弹窗：不继续后续操作
            return;
        }

        const res = await mongoApi.deleteDocs.request({
            id: tab.mongoId,
            database: tab.database,
            collection: tab.collection,
            idTokens: [doc.idToken],
        });
        if (res.deletedCount < 1) {
            Msg.warning('mongo.docAlreadyGone');
        } else {
            Msg.deleteSuccess();
        }
        await onChanged();
    }

    return { state, typedSummary, openAdd, openEdit, close, save, remove };
}
