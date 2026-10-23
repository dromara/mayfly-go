import { storeToRefs } from 'pinia';
import { ref } from 'vue';

import { useThemeConfig } from '@/store/themeConfig';

/**
 * 壁纸 base64 字符预算：实测 Chrome 对内联自定义属性值有 ~2Mi(2,097,152)字符的静默丢弃上限
 * (setProperty 超限不报错、变量直接消失 —— 正是「替换壁纸完全没反应」的根因)，
 * 同时留足 localStorage 配额余量(单键 5MB 总量)。1.4M 字符 ≈ 1MB 二进制，壁纸素材绰绰有余
 */
export const WALLPAPER_BUDGET = 1_400_000;

/** 内置玻璃壁纸预设（渐变定义见 theme/glass/wallpaper.scss，缩略图直接复用同名 CSS 变量） */
export const WALLPAPERS = [
    { id: 'none', nameKey: 'layout.config.wallpaperSpots', cssVar: '--wallpaper-spots' },
    { id: 'aurora', nameKey: 'layout.config.wallpaperAurora', cssVar: '--wallpaper-aurora' },
    { id: 'dusk', nameKey: 'layout.config.wallpaperDusk', cssVar: '--wallpaper-dusk' },
    { id: 'reef', nameKey: 'layout.config.wallpaperReef', cssVar: '--wallpaper-reef' },
    { id: 'bloom', nameKey: 'layout.config.wallpaperBloom', cssVar: '--wallpaper-bloom' },
    { id: 'nebula', nameKey: 'layout.config.wallpaperNebula', cssVar: '--wallpaper-nebula' },
] as const;

/** 按给定尺寸/质量编码一次(优先 webp, 不支持则 jpeg) */
export function encodeWallpaper(img: HTMLImageElement, maxEdge: number, quality: number): string {
    const scale = Math.min(1, maxEdge / Math.max(img.width, img.height));
    const canvas = document.createElement('canvas');
    canvas.width = Math.max(1, Math.round(img.width * scale));
    canvas.height = Math.max(1, Math.round(img.height * scale));
    const ctx = canvas.getContext('2d');
    if (!ctx) return '';
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    const webp = canvas.toDataURL('image/webp', quality);
    return webp.startsWith('data:image/webp') ? webp : canvas.toDataURL('image/jpeg', quality);
}

/**
 * 壁纸压缩(字节预算制)：高熵素材(噪点/细节密集照片)在固定 2560+q0.82 下仍可产出 2M+ 字符
 * 触顶被丢弃, 必须逐级降分辨率+质量直到落进预算。玻璃壁纸经 blur 采样, 1280px 底档无明显画质损失
 */
export function compressWallpaper(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const objectUrl = URL.createObjectURL(file);
        const img = new Image();
        img.onload = () => {
            URL.revokeObjectURL(objectUrl);
            for (const [edge, q] of [
                [2560, 0.82],
                [1920, 0.72],
                [1440, 0.62],
                [1280, 0.5],
            ] as const) {
                const dataUrl = encodeWallpaper(img, edge, q);
                if (dataUrl && dataUrl.length <= WALLPAPER_BUDGET) {
                    resolve(dataUrl);
                    return;
                }
            }
            // 终极兜底：最低档直接用(再小的编码产出也远超照片壁纸需求)
            const fallback = encodeWallpaper(img, 1080, 0.45);
            fallback ? resolve(fallback) : reject(new Error('wallpaper encode failed'));
        };
        img.onerror = () => {
            URL.revokeObjectURL(objectUrl);
            reject(new Error('image decode failed'));
        };
        img.src = objectUrl;
    });
}

export function readFileAsDataUrl(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = (e) => resolve(e.target?.result as string);
        reader.onerror = () => reject(reader.error);
        reader.readAsDataURL(file);
    });
}

/**
 * 壁纸交互域：内置壁纸选择 / 磨砂滑杆 / 自定义上传与清除。
 * SFC 只负责模板绑定，主题配置的读写全部收在这里（上传链路含重编码与预算控制）。
 */
export function useWallpaperUpload(setDispatchThemeConfig: () => void) {
    const themeConfigStore = useThemeConfig();
    const { themeConfig } = storeToRefs(themeConfigStore);

    const bgUploadRef = ref();

    // 选择内置壁纸
    const onWallpaperSelect = (id: string) => {
        themeConfigStore.setGlassWallpaper(id);
        setDispatchThemeConfig();
    };

    // 磨砂程度滑杆: 0=纹理清晰(默认零磨砂), store 内联覆盖 --glass-fx-chrome/panel/content 档位
    const onFrostChange = (v: number | number[]) => {
        themeConfigStore.setGlassFrost(Array.isArray(v) ? v[0] : v);
        setDispatchThemeConfig();
    };

    // 触发隐藏的上传器
    const triggerUpload = () => {
        const input = (bgUploadRef.value?.$el as HTMLElement | undefined)?.querySelector('input[type="file"]') as HTMLInputElement | null;
        input?.click();
    };

    // 点击「自定义」瓦片：已有图片则直接应用，否则打开文件选择
    const onCustomWallpaperClick = () => {
        if (themeConfig.value.bgImage) {
            themeConfigStore.setGlassWallpaper('custom');
            setDispatchThemeConfig();
        } else {
            triggerUpload();
        }
    };

    // 背景图上传：统一走字节预算编码链路；gif/svg 在预算内直存(保留动画/矢量)，超预算同样重编码
    const onBgImageUpload = async (uploadFile: any) => {
        const file = uploadFile.raw;
        if (!file) return;
        let dataUrl = '';
        try {
            const raw = await readFileAsDataUrl(file);
            const keepNative = (file.type === 'image/gif' || file.type === 'image/svg+xml') && raw.length <= WALLPAPER_BUDGET;
            dataUrl = keepNative ? raw : await compressWallpaper(file);
        } catch {
            // 兜底：canvas 链路异常(gif 解码失败/画布不可用等)时回退原文件直读，能否生效取决于体积
            try {
                dataUrl = await readFileAsDataUrl(file);
            } catch {
                return;
            }
        }
        themeConfigStore.setBgImage(dataUrl);
        themeConfigStore.setGlassWallpaper('custom');
        setDispatchThemeConfig();
    };

    const onBgImageClear = () => {
        themeConfigStore.setBgImage('');
        themeConfigStore.setGlassWallpaper('none');
        setDispatchThemeConfig();
    };

    return {
        bgUploadRef,
        wallpapers: WALLPAPERS,
        onWallpaperSelect,
        onFrostChange,
        triggerUpload,
        onCustomWallpaperClick,
        onBgImageUpload,
        onBgImageClear,
    };
}
