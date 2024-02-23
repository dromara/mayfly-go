import { saveThemeConfig } from '@/common/utils/storage';
import { isDark } from './user.vue';

export const switchDark = () => {
    themeConfig.value.isDark = isDark.value;
    if (isDark.value) {
        themeConfig.value.editorTheme = 'vs-dark';
    } else {
        themeConfig.value.editorTheme = 'vs';
    }
    // 如果终端主题不是自定义主题，则切换主题
    if (themeConfig.value.terminalTheme != 'custom') {
        if (isDark.value) {
            themeConfig.value.terminalTheme = 'dark';
        } else {
            themeConfig.value.terminalTheme = 'solarizedLight';
        }
    }
    saveThemeConfig(themeConfig.value);
};
