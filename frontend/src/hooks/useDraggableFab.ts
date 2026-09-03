/**
 * useDraggableFab - 悬浮球拖拽定位 composable
 *
 * 从 GlobalNotificationFab 提炼的通用能力：右下角悬浮球可拖拽移动、
 * 边界钳制、位置 localStorage 持久化、拖拽与点击区分（hasMoved）。
 *
 * 消费方（GlobalNotificationFab / AiAssistantFab）只负责渲染与点击语义，
 * 几何与持久化全部收敛于此。
 */
import { onMounted, onUnmounted, ref } from 'vue';

export interface FabPosition {
    bottom: number;
    right: number;
}

export interface UseDraggableFabOptions {
    /** localStorage 存储 key（各消费方独立，避免互相覆盖） */
    storageKey: string;
    /** 默认位置（距视口 bottom/right 像素） */
    defaultPosition?: FabPosition;
    /** 拖拽出界时按钮至少保留的可见像素 */
    minVisible?: number;
    /** 位移超过该像素视为拖拽而非点击 */
    dragThreshold?: number;
}

export function useDraggableFab(options: UseDraggableFabOptions) {
    const { storageKey, defaultPosition = { bottom: 20, right: 20 }, minVisible = 50, dragThreshold = 3 } = options;

    const position = ref<FabPosition>({ ...defaultPosition });
    const isDragging = ref(false);
    /** 本次按下到松开之间是否发生了拖拽位移（用于抑制误触 click） */
    const hasMoved = ref(false);
    const dragStart = ref({ x: 0, y: 0, initialBottom: 0, initialRight: 0 });

    const startDrag = (event: MouseEvent) => {
        // 只在左键拖拽时生效
        if (event.button !== 0) return;

        isDragging.value = true;
        hasMoved.value = false;
        dragStart.value = {
            x: event.clientX,
            y: event.clientY,
            initialBottom: position.value.bottom,
            initialRight: position.value.right,
        };

        document.addEventListener('mousemove', onDrag);
        document.addEventListener('mouseup', stopDrag);

        // 防止拖拽时选中文本
        document.body.style.userSelect = 'none';
    };

    const onDrag = (event: MouseEvent) => {
        if (!isDragging.value) return;

        const deltaY = event.clientY - dragStart.value.y;
        const deltaX = event.clientX - dragStart.value.x;

        // 如果移动距离超过阈值，认为是拖拽而不是点击
        if (Math.abs(deltaX) > dragThreshold || Math.abs(deltaY) > dragThreshold) {
            hasMoved.value = true;
        }

        // 更新位置（注意：鼠标向下移动时 bottom 应该减小）
        position.value.bottom = dragStart.value.initialBottom - deltaY;
        position.value.right = dragStart.value.initialRight - deltaX;

        // 确保不会移出屏幕（留出至少 minVisible 保证按钮可见）
        const windowHeight = window.innerHeight;
        const windowWidth = window.innerWidth;
        if (position.value.bottom < 0) position.value.bottom = 0;
        if (position.value.right < 0) position.value.right = 0;
        if (position.value.bottom > windowHeight - minVisible) position.value.bottom = windowHeight - minVisible;
        if (position.value.right > windowWidth - minVisible) position.value.right = windowWidth - minVisible;

        // 如果发生了移动，阻止默认行为
        if (hasMoved.value) {
            event.preventDefault();
        }
    };

    const stopDrag = () => {
        if (!isDragging.value) return;
        isDragging.value = false;
        document.removeEventListener('mousemove', onDrag);
        document.removeEventListener('mouseup', stopDrag);

        // 恢复文本选择
        document.body.style.userSelect = '';

        savePosition();
    };

    const savePosition = () => {
        try {
            localStorage.setItem(storageKey, JSON.stringify(position.value));
        } catch (error) {
            console.warn('Failed to save fab position:', error);
        }
    };

    const loadPosition = () => {
        try {
            const saved = localStorage.getItem(storageKey);
            if (saved) {
                const parsed = JSON.parse(saved) as FabPosition;
                // 验证数据有效性
                if (typeof parsed.bottom === 'number' && typeof parsed.right === 'number') {
                    position.value = parsed;
                }
            }
        } catch (error) {
            console.warn('Failed to load fab position:', error);
        }
    };

    // 组件卸载时清理事件监听
    onUnmounted(() => {
        document.removeEventListener('mousemove', onDrag);
        document.removeEventListener('mouseup', stopDrag);
        document.body.style.userSelect = '';
    });

    onMounted(() => {
        loadPosition();
    });

    return { position, isDragging, hasMoved, startDrag };
}
