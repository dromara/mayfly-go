import EnumValue from '@/common/Enum';
import { i18n } from '@/i18n';

export const NodeTypeEnum = {
    Start: EnumValue.of('start', i18n.global.t('flow.start')).setExtra({
        order: 1,
        text: i18n.global.t('flow.start'),
        defaultProp: {},
    }),

    End: EnumValue.of('end', i18n.global.t('flow.end')).setExtra({
        order: 100,
        text: i18n.global.t('flow.end'),
        defaultProp: {},
    }),

    Edge: EnumValue.of('flow-edge', i18n.global.t('flow.flowEdge')).setExtra({
        text: i18n.global.t('flow.flowEdge'),
    }),

    UserTask: EnumValue.of('usertask', i18n.global.t('flow.usertask')).setExtra({
        order: 2,
        type: 'usertask',
        text: i18n.global.t('flow.usertask'),
    }),

    Serial: EnumValue.of('serial', i18n.global.t('flow.serial')).setExtra({
        order: 3,
        text: i18n.global.t('flow.serial'),
        defaultProp: { condition: `{{ procinstTaskStatus == 1 }}` },
    }),

    Parallel: EnumValue.of('parallel', i18n.global.t('flow.parallel')).setExtra({
        order: 4,
        text: i18n.global.t('flow.parallel'),
        defaultProp: {},
    }),
};
