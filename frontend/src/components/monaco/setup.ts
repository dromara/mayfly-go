/**
 * monaco 编辑器装配入口 —— 项目内所有 monaco 运行时导入都必须走本文件
 *
 * ## 为什么需要这一层
 *
 * `from 'monaco-editor'` 命中的是包入口 esm/vs/index.js：一次性注册 81 种语言的语法着色、
 * 全部编辑器功能，以及 LSP 客户端（monaco.lsp）。实测语言 id 共 91 个，本项目只用到 13 个
 * （12 个显式装配 + monaco 自带的 plaintext），且零引用 LSP。
 * 历史实现里 MonacoEditor.vue 一边手写按需注册语言与功能，一边又 `import * as monaco from 'monaco-editor'`，
 * 按需注册的意图被桶导入完全抵消，于是每个用到编辑器的页面都要下载整包。
 *
 * ## 语言
 *
 * 取值来源为 MonacoEditor.vue 的 languageArr、ai/integration/utils.ts 的 EXT_LANGUAGE_MAP、
 * MachineFileContent.vue 的 getFileType()、msg 的 TmplTypeEnum，以及各数据库方言编辑器（统一为 sql）。
 *
 * definitions 只提供语法着色与词法；features 额外提供 worker 驱动的语言服务（校验/格式化/悬浮/重命名）。
 * json 没有独立的 definitions，其语言注册就在 features/json/register.js 内，故必须整体引入。
 * 不装 css / typescript 语言服务：二者需要专属 worker，而 MonacoEditor.vue 的 getWorker 只提供
 * editor/json/html 三种（未知 label 会拿到 editor.worker，协议不匹配），即这两套语言服务在运行时本就失效；
 * css 等语言的 definitions 仍保留，着色不受影响。
 *
 * ## 编辑器功能：为什么是「全量」而不是逐项挑选
 *
 * 曾按 provider 缺席逐项剔除过 codelens/inlayHints/parameterHints/rename/semanticTokens/
 * inlineCompletions/diffEditor 等本应用不可能触发的功能，实测**产物字节完全不变**（chunk hash 一致）：
 * 任一语言服务的 workerManager 都会静态引入 esm/vs/internal/common/workers.js，
 * 而该文件（名为 common，实为浏览器侧）又静态引入了完整的 74 项 contrib。
 * 也就是说只要保留 json 语言（其注册与语言服务同入口），contrib 集合就无法裁剪，逐项挑只会徒增维护面。
 *
 * ## 维护约定
 *
 * - 新增语言或功能在本文件登记；业务文件禁止值导入 'monaco-editor'（纯类型可用 import type，
 *   已由 eslint no-restricted-imports 约束），__tests__/setup.test.ts 守住语言集合与核心功能注册。
 * - 想进一步瘦身只能从「去掉全部语言服务」或「换编辑器」入手：json 没有独立 definitions，
 *   它的词法着色与校验/格式化/补全都来自 features/json 这套语言服务，去掉即 JSON 编辑器变纯色文本。
 */

// ---------------- 编辑器功能（monaco 官方聚合入口，等价于 editor.main.js 的功能集） ----------------

import 'monaco-editor/features/codicon/register.js'; // 图标字体样式（聚合入口未含，且 css 文件无法经包 exports map 深引，只能走这个 js 壳）
import 'monaco-editor/features/register.all.js';

// register.all.js 未覆盖、但 editor.main.js 会额外注册的功能，缺一项即有静默失效的能力
import 'monaco-editor/editor/browser/coreCommands.js'; // 光标移动、删除、行列增删等基础编辑命令
import 'monaco-editor/editor/common/standaloneStrings.js'; // standalone 模式的界面文案（菜单/提示）
import 'monaco-editor/editor/contrib/caretOperations/browser/caretOperations.js'; // 光标整体左右移
import 'monaco-editor/editor/contrib/dropOrPasteInto/browser/copyPasteContribution.js'; // 粘贴智能处理
import 'monaco-editor/editor/contrib/find/browser/findController.js'; // 查找替换（Cmd/Ctrl+F）
import 'monaco-editor/editor/contrib/gotoSymbol/browser/goToCommands.js'; // 符号跳转与返回
import 'monaco-editor/editor/contrib/gotoError/browser/markerSelectionStatus.js'; // 在错误标记间跳转
import 'monaco-editor/editor/contrib/semanticTokens/browser/documentSemanticTokens.js'; // 语义化着色
import 'monaco-editor/editor/contrib/suggest/browser/suggestController.js'; // 联想补全（Ctrl+Space）

// ---------------- 语言 ----------------

import 'monaco-editor/languages/definitions/sql/register.js';
import 'monaco-editor/languages/definitions/shell/register.js';
import 'monaco-editor/languages/definitions/yaml/register.js';
import 'monaco-editor/languages/definitions/dockerfile/register.js';
import 'monaco-editor/languages/definitions/javascript/register.js';
import 'monaco-editor/languages/definitions/html/register.js';
import 'monaco-editor/languages/definitions/css/register.js';
import 'monaco-editor/languages/definitions/xml/register.js';
import 'monaco-editor/languages/definitions/python/register.js';
import 'monaco-editor/languages/definitions/markdown/register.js';
import 'monaco-editor/languages/definitions/java/register.js';

import 'monaco-editor/languages/features/json/register.js'; // json 语言的注册也在此入口内
import 'monaco-editor/languages/features/html/register.js';

// ---------------- 对外 API ----------------

export * from 'monaco-editor/editor/editor.api.js';
