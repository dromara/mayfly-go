window.globalConfig = {
    // 默认为空，以访问根目录为api请求地址。若前后端分离部署可单独配置该后端api请求地址
    BaseApiUrl: '',
    BaseWsUrl: '',
};

// index.html添加百秒级时间戳，防止被浏览器缓存
// !(function () {
//     let t = 't=' + new Date().getTime().toString().substring(0, 8);
//     let search = location.search;
//     let m = search && search.match(/t=\d*/g);

//     console.log(location);
//     if (m[0]) {
//         if (m[0] !== t) {
//             location.search = search.replace(m[0], t);
//         }
//     } else {
//         if (search.indexOf('?') > -1) {
//             location.search = search + '&' + t;
//         } else {
//             location.search = t;
//         }
//     }
// })();
