/* eslint-disable */
import {IDisposable} from 'monaco-editor';

declare module '*.vue' {
	import type { DefineComponent } from 'vue';
	const component: DefineComponent<{}, {}, any>;
	export default component;
}


declare global {
	interface Window {
		completionItemProvider?: IDisposable | undefined;
	}
}


declare module 'sql-formatter';
declare module 'jsoneditor';
declare module 'asciinema-player';
declare module 'monaco-editor';
