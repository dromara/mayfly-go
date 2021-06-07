import type { App } from 'vue';

// 按钮波浪指令
export function wavesDirective(app: App) {
    app.directive('waves', {
        mounted(el, binding) {
            el.classList.add('waves-effect');
            binding.value && el.classList.add('waves-' + binding.value);
            function setConvertStyle(obj: any) {
                let style: string = '';
                for (let i in obj) {
                    if (obj.hasOwnProperty(i)) style += `${i}:${obj[i]};`;
                }
                return style;
            }
            function onCurrentClick(e: any) {
                let elDiv = document.createElement('div');
                elDiv.classList.add('waves-ripple');
                el.appendChild(elDiv);
                let styles = {
                    left: `${e.layerX}px`,
                    top: `${e.layerY}px`,
                    opacity: 1,
                    transform: `scale(${(el.clientWidth / 100) * 10})`,
                    'transition-duration': `750ms`,
                    'transition-timing-function': `cubic-bezier(0.250, 0.460, 0.450, 0.940)`,
                };
                elDiv.setAttribute('style', setConvertStyle(styles));
                setTimeout(() => {
                    elDiv.setAttribute(
                        'style',
                        setConvertStyle({
                            opacity: 0,
                            transform: styles.transform,
                            left: styles.left,
                            top: styles.top,
                        })
                    );
                    setTimeout(() => {
                        elDiv && el.removeChild(elDiv);
                    }, 750);
                }, 450);
            }
            el.addEventListener('mousedown', onCurrentClick, false);
        },
        unmounted(el) {
            el.addEventListener('mousedown', () => { });
        },
    });
}
