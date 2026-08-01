import { EnumValue } from '@/common/Enum';

export const ContainerStateEnum = {
    Running: EnumValue.of('running', 'docker.running').tagTypeSuccess().setExtra({ icon: 'VideoPlay' }),
    Stop: EnumValue.of('exited', 'docker.stopped').tagTypeDanger().setExtra({ icon: 'VideoPause' }),
};

export const ImageStateEnum = {
    Used: EnumValue.of(1, '已使用').tagTypeSuccess(),
    UnUsed: EnumValue.of(0, '未使用').tagTypeInfo(),
};
