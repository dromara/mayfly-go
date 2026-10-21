import { i18n } from '@/i18n';
import dayjs from 'dayjs';

const t = i18n.global.t;

/**
 * 首页动态里时间口径：
 * 1 小时内给相对时间（一眼看出新旧），当天/昨天给「今天 HH:mm」，更早给绝对日期。
 * 不引入 dayjs relativeTime 插件：多语言文案仍要走项目 i18n 词条，避免出现英文后缀混进中文界面
 */
export function formatFeedTime(value?: string | Date | null): string {
    if (!value) {
        return '-';
    }
    const target = dayjs(value);
    if (!target.isValid()) {
        return '-';
    }

    const now = dayjs();
    const diffMinutes = now.diff(target, 'minute');
    if (diffMinutes < 1) {
        return t('home.justNow');
    }
    if (diffMinutes < 60) {
        return t('home.minutesAgo', { n: diffMinutes });
    }
    if (target.isSame(now, 'day')) {
        return `${t('home.today')} ${target.format('HH:mm')}`;
    }
    if (target.isSame(now.subtract(1, 'day'), 'day')) {
        return `${t('home.yesterday')} ${target.format('HH:mm')}`;
    }
    if (target.isSame(now, 'year')) {
        return target.format('MM-DD HH:mm');
    }
    return target.format('YYYY-MM-DD');
}

/**
 * 当前时段的问候语 i18n key。
 * 分界点按运维作息取：5-11 早上、11-13 中午、13-18 下午、18-23 晚上、其余凌晨
 */
export function greetingKeyOf(date: Date = new Date()): string {
    const hour = date.getHours();
    if (hour >= 5 && hour < 11) {
        return 'home.greetingMorning';
    }
    if (hour >= 11 && hour < 13) {
        return 'home.greetingNoon';
    }
    if (hour >= 13 && hour < 18) {
        return 'home.greetingAfternoon';
    }
    if (hour >= 18 && hour < 23) {
        return 'home.greetingEvening';
    }
    return 'home.greetingDawn';
}
