import { nextTick } from 'vue';
import '@/theme/loading.scss';
import { useThemeConfig } from '@/store/themeConfig';
import { storeToRefs } from 'pinia';

/**
 * 页面全局 Loading
 * @method start 创建 loading
 * @method done 移除 loading
 */
export const NextLoading = {
    // 创建 loading
    start: () => {
        // 如果已经存在loading元素，则不重复创建
        if (document.querySelector('.loading-next')) {
            return;
        }

        const bodys: Element = document.body;
        const div = <HTMLElement>document.createElement('div');
        div.setAttribute('class', 'loading-next');

        const { themeConfig } = storeToRefs(useThemeConfig());
        if (themeConfig.value.isDark) {
            div.classList.add('dark');
        }

        const htmls = `
            <div class="loading-next-box">
                <div class="loading-next-box-warp">
                    <div class="loading-next-box-item"></div>
                    <div class="loading-next-box-item"></div>
                    <div class="loading-next-box-item"></div>
                    <div class="loading-next-box-item"></div>
                    <div class="loading-next-box-item"></div>
                    <div class="loading-next-box-item"></div>
                    <div class="loading-next-box-item"></div>
                    <div class="loading-next-box-item"></div>
                    <div class="loading-next-box-item"></div>
                </div>
            </div>
        `;
        div.innerHTML = htmls;

        // 插入到body的第一个子元素之前，避免影响布局
        if (bodys.firstChild) {
            bodys.insertBefore(div, bodys.firstChild);
        } else {
            bodys.appendChild(div);
        }
    },
    // 移除 loading
    done: (time: number = 500) => {
        nextTick(() => {
            setTimeout(() => {
                const el = <HTMLElement>document.querySelector('.loading-next');
                if (el) {
                    // 添加淡出效果
                    el.style.transition = 'opacity 0.3s ease-out';
                    el.style.opacity = '0';
                    setTimeout(() => {
                        el?.parentNode?.removeChild(el);
                    }, 300);
                }
            }, time);
        });
    },
};

export function sleep(ms: number) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}
