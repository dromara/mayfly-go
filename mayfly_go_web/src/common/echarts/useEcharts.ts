import * as echarts from 'echarts'

export default function(dom: any, theme: any = null,  option: any) {
    let chart = echarts.init(dom, theme);
    chart.setOption(option);
    return chart;
}