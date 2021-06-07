import { getLocal } from '@/common/utils/storage';

// 全局组件大小
export const globalComponentSize = getLocal('themeConfig')?.globalComponentSize;
