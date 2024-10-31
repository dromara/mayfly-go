import { nextTick } from 'vue';
import '@/theme/loading.scss';

/**
 * 页面全局 Loading
 * @method start 创建 loading
 * @method done 移除 loading
 */
export const NextLoading = {
    // 创建 loading
    start: () => {
        const bodys: Element = document.body;
        const div = <HTMLElement>document.createElement('div');
        div.setAttribute('class', 'loading-next');
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
        bodys.insertBefore(div, bodys.childNodes[0]);
    },
    // 移除 loading
    done: (time: number = 1000) => {
        nextTick(() => {
            setTimeout(() => {
                const el = <HTMLElement>document.querySelector('.loading-next');
                el?.parentNode?.removeChild(el);
            }, time);
        });
    },
};

export function sleep(ms: number) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}
