import { Enum } from '@/common/Enum';

export default {
    // uri请求方法
    scriptTypeEnum: new Enum().add('RESULT', '有结果', 1).add('NO_RESULT', '无结果', 2).add('REAL_TIME', '实时交互', 3),
    // 文件类型枚举
    FileTypeEnum: new Enum().add('DIRECTORY', '目录', 1).add('FILE', '文件', 2),
}