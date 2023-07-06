import { getUseWatermark4Session, getUserInfo4Session } from '@/common/utils/storage';
import { dateFormat2 } from '@/common/utils/date';

// 页面添加水印效果
const setWatermark = (str: any) => {
    const id = '1.23452384164.123412416';
    if (document.getElementById(id) !== null) document.body.removeChild(document.getElementById(id) as any);
    const can = document.createElement('canvas');
    can.width = 400;
    can.height = 250;
    const cans: any = can.getContext('2d');
    cans.rotate((-20 * Math.PI) / 180);
    cans.font = '14px Vedana';
    cans.fillStyle = 'rgba(200, 200, 200, 0.35)';
    cans.textAlign = 'left';
    cans.textBaseline = 'Middle';
    // cans.fillText('mayfly go', can.width / 4, can.height )
    cans.fillText(str, can.width / 8, can.height / 2);

    const div = document.createElement('div');
    div.id = id;
    div.style.pointerEvents = 'none';
    div.style.top = '30px';
    div.style.left = '0px';
    div.style.position = 'fixed';
    div.style.zIndex = '10000000';
    div.style.width = document.documentElement.clientWidth + 'px';
    div.style.height = document.documentElement.clientHeight + 'px';
    div.style.background = `url(${can.toDataURL('image/png')}) left top repeat`;
    document.body.appendChild(div);
    return id;
};

function set(str: any) {
    let id = setWatermark(str);
    if (document.getElementById(id) === null) id = setWatermark(str);
}

function del() {
    let id = '1.23452384164.123412416';
    if (document.getElementById(id) !== null) document.body.removeChild(document.getElementById(id) as any);
}

const watermark = {
    use: () => {
        setTimeout(() => {
            const userinfo = getUserInfo4Session();
            if (userinfo && getUseWatermark4Session()) {
                set(`${userinfo.username} ${dateFormat2('yyyy-MM-dd HH:mm:ss', new Date())}`);
            } else {
                del();
            }
        }, 1500);
    },
    // 设置水印
    set: (str: any) => {
        set(str);
    },
    // 删除水印
    del: () => {
        del();
    },
};

export default watermark;
