import { ref } from 'vue';

const vw = ref(document.documentElement.clientWidth);
const vh = ref(document.documentElement.clientHeight);

window.addEventListener('resize', () => {
    vw.value = document.documentElement.clientWidth;
    vh.value = document.documentElement.clientHeight;
});

/**
 * 获取视图宽高
 * @returns 视图宽高
 */
export function useViewport() {
    return { vw, vh };
}
