const allSvgIcons = import.meta.glob('./**/*.svg', { eager: true, as: 'raw' });

const iconNames = [];

/**
 * 获取本地图标
 * @returns 本地图标
 */
export function getLocalIcons() {
    return iconNames;
}

function convertSvgToSymbol(svgString, symbolId) {
    // 创建一个 DOMParser 实例
    const parser = new DOMParser();
    // 解析 SVG 字符串为文档对象
    const doc = parser.parseFromString(svgString, 'image/svg+xml');
    // 获取外层的 <svg> 元素
    const svgElement = doc.querySelector('svg');
    // 创建一个新的 <symbol> 元素
    const symbolElement = document.createElementNS('http://www.w3.org/2000/svg', 'symbol');
    // 设置 <symbol> 元素的 id 属性
    symbolElement.setAttribute('id', symbolId);
    // 复制 <svg> 元素的 viewBox 属性到 <symbol> 元素
    if (svgElement.hasAttribute('viewBox')) {
        symbolElement.setAttribute('viewBox', svgElement.getAttribute('viewBox'));
    }
    // 将 <svg> 元素的所有子节点复制到 <symbol> 元素中
    while (svgElement.firstChild) {
        symbolElement.appendChild(svgElement.firstChild);
    }
    // 创建一个临时的 div 元素来存储 <symbol> 元素的内容
    const tempDiv = document.createElement('div');
    tempDiv.appendChild(symbolElement);
    // 返回 <symbol> 标签的内容
    return tempDiv.innerHTML;
}

// iconfont 代码
(function (c) {
    let svgsymbols = '<svg>';
    // 初始化icons
    for (const path in allSvgIcons) {
        // ./df/input.svg
        // 转为 df/input
        const name = path.replace('.svg', '').replace(/^\.\//, '');
        iconNames.push(`icon ${name}`);
        svgsymbols += convertSvgToSymbol(allSvgIcons[path], name);
    }
    svgsymbols += '</svg>';

    var t = (t = document.getElementsByTagName('script'))[t.length - 1],
        a = t.getAttribute('data-injectcss'),
        t = t.getAttribute('data-disable-injectsvg');
    if (!t) {
        var l,
            e,
            i,
            o,
            n,
            h = function (t, a) {
                a.parentNode.insertBefore(t, a);
            };
        if (a && !c.__iconfont__svg__cssinject__) {
            c.__iconfont__svg__cssinject__ = !0;
            try {
                document.write(
                    '<style>.svgfont {display: inline-block;width: 1em;height: 1em;fill: currentColor;vertical-align: -0.1em;font-size:16px;}</style>'
                );
            } catch (t) {
                console && console.log(t);
            }
        }
        (l = function () {
            var t,
                a = document.createElement('div');
            (a.innerHTML = svgsymbols),
                (a = a.getElementsByTagName('svg')[0]) &&
                    (a.setAttribute('aria-hidden', 'true'),
                    (a.style.position = 'absolute'),
                    (a.style.width = 0),
                    (a.style.height = 0),
                    (a.style.overflow = 'hidden'),
                    (a = a),
                    (t = document.body).firstChild ? h(a, t.firstChild) : t.appendChild(a));
        }),
            document.addEventListener
                ? ~['complete', 'loaded', 'interactive'].indexOf(document.readyState)
                    ? setTimeout(l, 0)
                    : ((e = function () {
                          document.removeEventListener('DOMContentLoaded', e, !1), l();
                      }),
                      document.addEventListener('DOMContentLoaded', e, !1))
                : document.attachEvent &&
                  ((i = l),
                  (o = c.document),
                  (n = !1),
                  s(),
                  (o.onreadystatechange = function () {
                      'complete' == o.readyState && ((o.onreadystatechange = null), d());
                  }));
    }
    function d() {
        n || ((n = !0), i());
    }
    function s() {
        try {
            o.documentElement.doScroll('left');
        } catch (t) {
            return void setTimeout(s, 50);
        }
        d();
    }
})(window);
