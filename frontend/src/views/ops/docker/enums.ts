import { EnumValue } from '@/common/Enum';

export const ContainerStateEnum = {
    Running: EnumValue.of('running', 'docker.running').tagTypeSuccess().setExtra({ icon: 'VideoPlay' }),
    Stop: EnumValue.of('exited', 'docker.stopped').tagTypeDanger().setExtra({ icon: 'VideoPause' }),
};

export const ImageStateEnum = {
    Used: EnumValue.of(true, '已使用').tagTypeSuccess(),
    UnUsed: EnumValue.of(false, '未使用').tagTypeInfo(),
};
