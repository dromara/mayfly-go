// https://www.npmjs.com/package/mitt
import mitt, { Emitter } from 'mitt';

// 类型
const emitter: Emitter<any> = mitt<any>();

// 导出
export default emitter;
