import { EnumValue } from '@/common/Enum';

export const ChannelStatusEnum = {
    Enable: EnumValue.of(1, 'common.enable').tagTypeSuccess(),
    Disable: EnumValue.of(-1, 'common.disable').tagTypeDanger(),
};

export const TmplStatusEnum = {
    Enable: EnumValue.of(1, 'common.enable').tagTypeSuccess(),
    Disable: EnumValue.of(-1, 'common.disable').tagTypeDanger(),
};

export const TmplTypeEnum = {
    Text: EnumValue.of(1, 'text'),
    Markdown: EnumValue.of(2, 'markdown'),
    Html: EnumValue.of(3, 'html'),
};

export const ChannelTypeEnum = {
    Email: EnumValue.of('email', 'msg.email').setExtra({ component: 'ChannelEmail', msgTypes: [TmplTypeEnum.Text, TmplTypeEnum.Markdown, TmplTypeEnum.Html] }),
    DingBot: EnumValue.of('dingBot', 'msg.dingBot').setExtra({ component: 'ChannelDing', msgTypes: [TmplTypeEnum.Text, TmplTypeEnum.Markdown] }),
    QywxBot: EnumValue.of('qywxBot', 'msg.qywxBot').setExtra({ msgTypes: [TmplTypeEnum.Text.value, TmplTypeEnum.Markdown] }),
    FeishuBot: EnumValue.of('feishuBot', 'msg.feishuBot').setExtra({ component: 'ChannelDing', msgTypes: [TmplTypeEnum.Text] }),
};
