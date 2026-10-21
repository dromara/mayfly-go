/**
 * 提交数据（ChatInputSubmitData）的组装单点
 *
 * 为什么单独抽出：文本与段序列必须**成对**携带——芯片的定位信息（resourceType/id/ip/
 * authCertName…）只存在段里，只带 text 就等于把用户的引用抹掉（模型收到一句带字面
 * `@名字` 的文本，不知道要操作谁）。这个不变量此前在两处各自构造（聊天输入框提交、
 * 消息编辑重发），后者一度写成 `segments: []`，静态类型完全合法、门禁全绿，但业务
 * 语义已经错了。收成一处并配单测，才让「丢段」变成会被测出来的事。
 */
import type { ChipSegment } from './chipRegistry';
import type { ChatInputSubmitData } from './types';
import type { MessageAttachment } from '../protocol/types';

/**
 * 组装一次提交数据：文本按 trim 后携带，段序列原样透传（不去重、不过滤、不置空）。
 *
 * attachments 为空数组时归一为 undefined：避免下游对「空数组」与「无附件」两种假
 * 分支分别处理（历史上图片段与文件段的双路径不一致就是这么来的）
 */
export function buildSubmitData(
    text: string,
    segments: ChipSegment[] | undefined,
    attachments?: MessageAttachment[],
): ChatInputSubmitData {
    return {
        text: text.trim(),
        segments: segments ?? [],
        ...(attachments?.length ? { attachments } : {}),
    };
}
