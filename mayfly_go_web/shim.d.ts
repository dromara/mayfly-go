/* eslint-disable */
declare module '*.vue' {
	import type { DefineComponent } from 'vue';
	const component: DefineComponent<{}, {}, any>;
	export default component;
}
declare module 'codemirror';
declare module 'sql-formatter';
declare module 'jsoneditor';
declare module 'asciinema-player';