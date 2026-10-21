/**
 * 审批回执照别的线协议常量（前端侧镜像，唯一声明处）。
 *
 * 与后端 `server/internal/agentops/application/proposal_ack.go` 一一对应：后端 FlowBizHandle 的
 * 非终态返回被 flow 落进工单 `biz_handle_res`，这里判断「这条回执是有结论的结果，还是只是受理
 * 凭据」。两侧各抄一份字面量时，后端改一次措辞就会让审批面板从「等待提示」静默退化成裸直出 JSON
 * ——审批人看到一坨回执而不知道变更正在异步跑，且前后端都不报错。
 *
 * 因此本文件是组件与其测试的**共同引用点**（不要在测试里再写一遍字面量），并由
 * `agentRemediationBiz.test.ts` 钉住字面值；后端另有 `proposal_ack_test.go` 钉同一组值。
 * 任一侧改动都会红，红了才去看另一侧。
 */

/** 回执里表示「执行已投递、尚无结论」的位（值域见 ACK_EXECUTIONS） */
export const ACK_KEY_EXECUTION = 'execution';

/** 回执里表示「本次没有新动作（幂等落败/已处理）」的位（值域见 ACK_MESSAGES） */
export const ACK_KEY_MESSAGE = 'message';

/** 已受理：执行已排队（异步跑，终态稍后回写工单） */
export const ACK_EXECUTION_QUEUED = 'queued';

/** 崩溃重投：上一次领取后崩在执行之前，本次重新投递 */
export const ACK_EXECUTION_REQUEUED = 'requeued';

/** 提案已是终态，本次审批回调是重驱动 */
export const ACK_MESSAGE_ALREADY_PROCESSED = 'already_processed';

/** CAS 领取落败（并发审批/重复驱动） */
export const ACK_MESSAGE_ALREADY_CLAIMED = 'already_claimed';

/**
 * 历史字面值：回执值曾以英文句子形态落库（`already processed` / `already claimed`），
 * 存量工单的 biz_handle_res 里仍是这两串。改判为标识符形态后必须继续识别它们，
 * 否则老工单打开面板就是裸 JSON。新数据不会再产出这两串（后端只发上面的常量值），
 * 因此本清单只增不减、也不再被后端引用。
 */
export const LEGACY_ACK_MESSAGES = ['already processed', 'already claimed'];

/** 「已受理」形态全集：命中任一即降级为等待提示，不直出 JSON */
export const ACK_EXECUTIONS: string[] = [ACK_EXECUTION_QUEUED, ACK_EXECUTION_REQUEUED];

/** 「本次无新动作」形态全集（含历史句子形态） */
export const ACK_MESSAGES: string[] = [ACK_MESSAGE_ALREADY_PROCESSED, ACK_MESSAGE_ALREADY_CLAIMED, ...LEGACY_ACK_MESSAGES];
