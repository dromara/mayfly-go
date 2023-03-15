/* eslint-disable */
import {IDisposable} from 'monaco-editor';
declare global {
	interface Window {
		completionItemProvider?: IDisposable | undefined;
	}
}

// 申明外部 npm 插件模块
declare module 'sql-formatter';
declare module 'jsoneditor';
declare module 'asciinema-player';
declare module 'monaco-editor';
declare module 'vue-grid-layout';